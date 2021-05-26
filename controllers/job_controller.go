package controllers

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"time"

	"github.com/go-logr/logr"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	clientbuilder "sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/ccremer/clustercode/api/v1alpha1"
	"github.com/ccremer/clustercode/builder"
	"github.com/ccremer/clustercode/cfg"
)

type (
	// JobReconciler reconciles Job objects
	JobReconciler struct {
		Client client.Client
		Log    logr.Logger
		Scheme *runtime.Scheme
	}
	// JobContext holds the parameters of a single reconciliation
	JobContext struct {
		ctx     context.Context
		job     *batchv1.Job
		jobType ClusterCodeJobType
		task    *v1alpha1.Task
		log     logr.Logger
	}
)

func (r *JobReconciler) SetupWithManager(mgr ctrl.Manager, l logr.Logger) error {
	if uncached, err := client.NewDelegatingClient(client.NewDelegatingClientInput{
		CacheReader: mgr.GetClient(),
		Client:      mgr.GetClient(),
		UncachedObjects: []client.Object{
			&batchv1.Job{},
		},
	}); err != nil {
		return err
	} else {
		r.Client = uncached
	}
	r.Scheme = mgr.GetScheme()
	r.Log = l
	pred, err := predicate.LabelSelectorPredicate(metav1.LabelSelector{MatchLabels: ClusterCodeLabels})
	if err != nil {
		return err
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&batchv1.Job{}, clientbuilder.WithPredicates(pred)).
		Complete(r)
}

// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete

func (r *JobReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	rc := &JobContext{
		job: &batchv1.Job{},
		ctx: ctx,
	}
	err := r.Client.Get(ctx, req.NamespacedName, rc.job)
	if err != nil {
		if apierrors.IsNotFound(err) {
			r.Log.V(1).Info("object not found, ignoring reconcile", "object", req.NamespacedName)
			return ctrl.Result{}, nil
		}
		r.Log.Error(err, "could not retrieve object", "object", req.NamespacedName)
		return ctrl.Result{Requeue: true, RequeueAfter: time.Minute}, err
	}
	rc.log = r.Log.WithValues("job", req.NamespacedName)
	if rc.job.GetDeletionTimestamp() != nil {
		rc.log.V(1).Info("job is being deleted, ignoring reconcile")
		return ctrl.Result{}, nil
	}
	jobType, err := rc.getJobType()
	if err != nil {
		rc.log.V(1).Info("cannot determine job type, ignoring reconcile", "error", err.Error())
		return ctrl.Result{}, nil
	}
	conditions := castConditions(rc.job.Status.Conditions)
	if !meta.IsStatusConditionPresentAndEqual(conditions, string(batchv1.JobComplete), metav1.ConditionTrue) {
		rc.log.V(1).Info("job is not completed yet, ignoring reconcile")
		return ctrl.Result{}, nil
	}
	rc.jobType = jobType
	switch jobType {
	case ClustercodeTypeSplit:
		return ctrl.Result{}, r.handleSplitJob(rc)
	case ClustercodeTypeCount:
		rc.log.Info("reconciled count job")
	case ClustercodeTypeSlice:
		rc.log.Info("reconciling slice job")
		return ctrl.Result{}, r.handleSliceJob(rc)
	case ClustercodeTypeMerge:
		rc.log.Info("reconciling merge job")
		return ctrl.Result{}, r.handleMergeJob(rc)
	}
	return ctrl.Result{}, nil
}

func (r *JobReconciler) handleSplitJob(rc *JobContext) error {
	rc.task = &v1alpha1.Task{}
	if err := r.Client.Get(rc.ctx, getOwner(rc.job), rc.task); err != nil {
		return err
	}

	return r.createCountJob(rc)
}

func (r *JobReconciler) handleSliceJob(rc *JobContext) error {
	indexStr, found := rc.job.Labels[ClustercodeSliceIndexLabelKey]
	if !found {
		return fmt.Errorf("cannot determine slice index, missing label '%s'", ClustercodeSliceIndexLabelKey)
	}
	index, err := strconv.Atoi(indexStr)
	if err != nil {
		return fmt.Errorf("cannot determine slice index from label '%s': %w", ClustercodeSliceIndexLabelKey, err)
	}

	rc.task = &v1alpha1.Task{}
	if err := r.Client.Get(rc.ctx, getOwner(rc.job), rc.task); err != nil {
		return err
	}
	finished := rc.task.Status.SlicesFinished
	finished = append(finished, v1alpha1.ClustercodeSliceRef{
		SliceIndex: index,
		JobName:    rc.job.Name,
	})
	rc.task.Status.SlicesFinished = finished
	rc.task.Status.SlicesFinishedCount = len(finished)

	var scheduled []v1alpha1.ClustercodeSliceRef
	for _, ref := range rc.task.Status.SlicesScheduled {
		if ref.SliceIndex != index {
			scheduled = append(scheduled, ref)
		}
	}
	rc.task.Status.SlicesScheduled = scheduled
	return r.Client.Status().Update(rc.ctx, rc.task)
}

func (r *JobReconciler) createCountJob(rc *JobContext) error {

	taskId := rc.task.Spec.TaskId
	cb := builder.NewContainerBuilder("clustercode").
		WithImage(cfg.Config.Operator.ClustercodeContainerImage).
		WithArgs("-v").
		AddArgs("--namespace=%s", rc.job.Namespace).
		AddArgs("count").
		AddArgs("--count.task-name=%s", rc.task.Name).
		Build()

	pb := builder.NewPodSpecBuilder(cb).Build()
	pvc := rc.task.Spec.Storage.IntermediatePvc
	pb.AddPvcMount(nil, IntermediateSubMountPath, pvc.ClaimName, filepath.Join("/clustercode", IntermediateSubMountPath), pvc.SubPath)
	pb.PodSpec.ServiceAccountName = rc.job.Spec.Template.Spec.ServiceAccountName
	pb.PodSpec.RestartPolicy = corev1.RestartPolicyNever

	job := &batchv1.Job{
		Spec: batchv1.JobSpec{
			BackoffLimit: pointer.Int32Ptr(0),
			Template: corev1.PodTemplateSpec{
				Spec: *pb.PodSpec,
			},
		},
	}
	builder.NewMetaBuilderWith(job).
		WithNamespace(rc.job.Namespace).
		WithName(fmt.Sprintf("%.*s-%s", 62-len(ClustercodeTypeCount), taskId, ClustercodeTypeCount)).
		WithLabels(ClusterCodeLabels, ClustercodeTypeCount.AsLabels(), taskId.AsLabels())

	if err := controllerutil.SetControllerReference(rc.task, job.GetObjectMeta(), r.Scheme); err != nil {
		rc.log.Info("could not set controller reference, deleting the task won't delete the job", "err", err.Error())
	}
	if err := r.Client.Create(rc.ctx, job); err != nil {
		if apierrors.IsAlreadyExists(err) {
			rc.log.Info("skip creating job, it already exists", "job", job.Name)
		} else {
			rc.log.Error(err, "could not create job", "job", job.Name)
			return err
		}
	} else {
		rc.log.Info("job created", "job", job.Name)
	}
	return nil
}

func (r *JobReconciler) handleMergeJob(rc *JobContext) error {
	rc.task = &v1alpha1.Task{}
	if err := r.Client.Get(rc.ctx, getOwner(rc.job), rc.task); err != nil {
		return err
	}

	return r.createCleanupJob(rc)
}

func (r *JobReconciler) createCleanupJob(rc *JobContext) error {

	taskId := rc.task.Spec.TaskId
	cb := builder.NewContainerBuilder("clustercode").
		WithImage(cfg.Config.Operator.ClustercodeContainerImage).
		WithImagePullPolicy(corev1.PullIfNotPresent).
		WithArgs("-v").
		AddArgs("--namespace=%s", rc.job.Namespace).
		AddArg("cleanup").
		AddArgs("--cleanup.task-name=%s", rc.task.Name).
		Build()

	pb := builder.NewPodSpecBuilder(cb)
	pb.AddPvcMount(nil, SourceSubMountPath, rc.task.Spec.Storage.SourcePvc.ClaimName, filepath.Join("/clustercode", SourceSubMountPath), rc.task.Spec.Storage.SourcePvc.SubPath)
	pb.AddPvcMount(nil, IntermediateSubMountPath, rc.task.Spec.Storage.IntermediatePvc.ClaimName, filepath.Join("/clustercode", IntermediateSubMountPath), rc.task.Spec.Storage.IntermediatePvc.SubPath)
	pb.PodSpec.ServiceAccountName = rc.job.Spec.Template.Spec.ServiceAccountName
	pb.PodSpec.SecurityContext = &corev1.PodSecurityContext{
		RunAsUser:  pointer.Int64Ptr(1000),
		RunAsGroup: pointer.Int64Ptr(0),
		FSGroup:    pointer.Int64Ptr(0),
	}
	pb.PodSpec.RestartPolicy = corev1.RestartPolicyNever
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%.*s-%s", 62-len(ClustercodeTypeCleanup), taskId, ClustercodeTypeCleanup),
			Namespace: rc.job.Namespace,
			Labels:    labels.Merge(ClusterCodeLabels, labels.Merge(ClustercodeTypeCleanup.AsLabels(), taskId.AsLabels())),
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: pointer.Int32Ptr(0),
			Template: corev1.PodTemplateSpec{
				Spec: *pb.Build().PodSpec,
			},
		},
	}
	if err := controllerutil.SetControllerReference(rc.task, job.GetObjectMeta(), r.Scheme); err != nil {
		rc.log.Info("could not set controller reference, deleting the task won't delete the job", "err", err.Error())
	}
	if err := r.Client.Create(rc.ctx, job); err != nil {
		if apierrors.IsAlreadyExists(err) {
			rc.log.Info("skip creating job, it already exists", "job", job.Name)
		} else {
			rc.log.Error(err, "could not create job", "job", job.Name)
			return err
		}
	} else {
		rc.log.Info("job created", "job", job.Name)
	}
	return nil
}

func (c JobContext) getJobType() (ClusterCodeJobType, error) {
	set := labels.Set(c.job.Labels)
	if !set.Has(ClustercodeTypeLabelKey) {
		return "", fmt.Errorf("missing label key '%s", ClustercodeTypeLabelKey)
	}
	label := set.Get(ClustercodeTypeLabelKey)
	for _, jobType := range ClustercodeTypes {
		if label == string(jobType) {
			return jobType, nil
		}
	}
	return "", fmt.Errorf("value of label '%s' unrecognized: %s", ClustercodeTypeLabelKey, label)
}

func castConditions(conditions []batchv1.JobCondition) (converted []metav1.Condition) {
	for _, c := range conditions {
		converted = append(converted, metav1.Condition{
			Type:               string(c.Type),
			Status:             metav1.ConditionStatus(c.Status),
			LastTransitionTime: c.LastTransitionTime,
			Reason:             c.Reason,
			Message:            c.Message,
		})
	}
	return converted
}
