package task

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"time"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/ccremer/clustercode/api/v1alpha1"
	"github.com/ccremer/clustercode/controllers"
	"github.com/ccremer/clustercode/controllers/pipeline"
)

type (
	// Reconciler reconciles Task objects
	Reconciler struct {
		Recorder record.EventRecorder
		*pipeline.ResourceAction
	}
	// ReconciliationContext holds the parameters of a single reconciliation
	ReconciliationContext struct {
		ctx       context.Context
		task      *v1alpha1.Task
		blueprint *v1alpha1.Blueprint
		Log       logr.Logger
	}
	TaskOpts struct {
		Args              []string
		JobType           controllers.ClusterCodeJobType
		MountSource       bool
		MountIntermediate bool
		MountTarget       bool
		MountConfig       bool
	}
)

func (r *Reconciler) SetupWithManager(mgr ctrl.Manager, l logr.Logger) error {
	r.ResourceAction = &pipeline.ResourceAction{
		Log:    l,
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}
	r.Recorder = mgr.GetEventRecorderFor("task-controller")
	pred, err := predicate.LabelSelectorPredicate(metav1.LabelSelector{MatchLabels: controllers.ClusterCodeLabels})
	if err != nil {
		return err
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Task{}, builder.WithPredicates(pred)).
		//Owns(&batchv1.Job{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(r)
}

// +kubebuilder:rbac:groups=clustercode.github.io,resources=tasks,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=clustercode.github.io,resources=tasks/status,verbs=get;update;patch

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	rc := &ReconciliationContext{
		ctx: ctx,
		task: &v1alpha1.Task{
			ObjectMeta: metav1.ObjectMeta{
				Name:      req.Name,
				Namespace: req.Namespace,
			},
		},
		Log: r.Log.WithValues("task", req.NamespacedName.String()),
	}

	splitJobPipeline := pipeline.NewPipeline(rc.Log).
		WithSteps(
			pipeline.NewStep("create split job", r.createSplitJob(rc)),
		)

	mergeJobPipeline := pipeline.NewPipeline(rc.Log).
		WithSteps(
			pipeline.NewStep("create merge job", r.createMergeJob(rc)),
		)

	result := pipeline.NewPipeline(rc.Log).
		WithSteps(
			pipeline.NewStep("get reconcile object", r.GetOrAbort(ctx, rc.task)),
			splitJobPipeline.AsNestedStep("split-job", splitJobPredicate(rc)),
			mergeJobPipeline.AsNestedStep("merge-job", mergeJobPredicate(rc)),
		).Run()
	if result.Err != nil {
		r.Log.Error(result.Err, "pipeline failed with error")
	}

	if result.Requeue {
		return ctrl.Result{RequeueAfter: 1 * time.Minute}, result.Err
	}
	rc.Log.Info("reconciled task")
	return ctrl.Result{}, result.Err
}

func (r *Reconciler) handleTask(rc *ReconciliationContext) error {
	if rc.task.Spec.SlicesPlannedCount == 0 {
		return r.createSplitJob(rc)
	}

	if len(rc.task.Status.SlicesFinished) >= rc.task.Spec.SlicesPlannedCount {
		return r.createMergeJob(rc)
	}
	// Todo: Check condition whether more jobs are needed
	nextSliceIndex := r.determineNextSliceIndex(rc)
	if nextSliceIndex < 0 {
		return nil
	} else {
		rc.Log.Info("scheduling next slice", "index", nextSliceIndex)
		return r.createSliceJob(rc, nextSliceIndex)
	}
}

func (r *Reconciler) determineNextSliceIndex(rc *ReconciliationContext) int {
	status := rc.task.Status
	if rc.task.Spec.ConcurrencyStrategy.ConcurrentCountStrategy != nil {
		maxCount := rc.task.Spec.ConcurrencyStrategy.ConcurrentCountStrategy.MaxCount
		if len(status.SlicesScheduled) >= maxCount {
			rc.Log.V(1).Info("reached concurrent max count, cannot schedule more", "max", maxCount)
			return -1
		}
	}
	for i := 0; i < rc.task.Spec.SlicesPlannedCount; i++ {
		if containsSliceIndex(status.SlicesScheduled, i) {
			continue
		}
		if containsSliceIndex(status.SlicesFinished, i) {
			continue
		}
		return i
	}
	return -1
}

func containsSliceIndex(list []v1alpha1.ClustercodeSliceRef, index int) bool {
	for _, t := range list {
		if t.SliceIndex == index {
			return true
		}
	}
	return false
}

func (r *Reconciler) createSplitJob(rc *ReconciliationContext) error {
	sourceMountRoot := filepath.Join("/clustercode", controllers.SourceSubMountPath)
	intermediateMountRoot := filepath.Join("/clustercode", controllers.IntermediateSubMountPath)
	variables := map[string]string{
		"${INPUT}":      filepath.Join(sourceMountRoot, rc.task.Spec.SourceUrl.GetPath()),
		"${OUTPUT}":     getSegmentFileNameTemplatePath(rc, intermediateMountRoot),
		"${SLICE_SIZE}": strconv.Itoa(rc.task.Spec.EncodeSpec.SliceSize),
	}
	job := CreateFfmpegJobDefinition(rc.task, &TaskOpts{
		Args:              MergeArgsAndReplaceVariables(variables, rc.task.Spec.EncodeSpec.DefaultCommandArgs, rc.task.Spec.EncodeSpec.SplitCommandArgs),
		JobType:           controllers.ClustercodeTypeSplit,
		MountSource:       true,
		MountIntermediate: true,
	})
	if err := controllerutil.SetControllerReference(rc.task, job.GetObjectMeta(), r.Scheme); err != nil {
		rc.Log.Info("could not set controller reference, deleting the task won't delete the job", "err", err.Error())
	}
	if err := r.Client.Create(rc.ctx, job); err != nil {
		if apierrors.IsAlreadyExists(err) {
			rc.Log.Info("skip creating job, it already exists", "job", job.Name)
		} else {
			rc.Log.Error(err, "could not create job", "job", job.Name)
		}
	} else {
		rc.Log.Info("job created", "job", job.Name)
	}
	return nil
}

func (r *Reconciler) createSliceJob(rc *ReconciliationContext, index int) error {
	intermediateMountRoot := filepath.Join("/clustercode", controllers.IntermediateSubMountPath)
	variables := map[string]string{
		"${INPUT}":  getSourceSegmentFileNameIndexPath(rc, intermediateMountRoot, index),
		"${OUTPUT}": getTargetSegmentFileNameIndexPath(rc, intermediateMountRoot, index),
	}
	job := CreateFfmpegJobDefinition(rc.task, &TaskOpts{
		Args:              MergeArgsAndReplaceVariables(variables, rc.task.Spec.EncodeSpec.DefaultCommandArgs, rc.task.Spec.EncodeSpec.TranscodeCommandArgs),
		JobType:           controllers.ClustercodeTypeSlice,
		MountIntermediate: true,
	})
	job.Name = fmt.Sprintf("%s-%d", job.Name, index)
	job.Labels[controllers.ClustercodeSliceIndexLabelKey] = strconv.Itoa(index)
	if err := controllerutil.SetControllerReference(rc.task, job.GetObjectMeta(), r.Scheme); err != nil {
		return fmt.Errorf("could not set controller reference: %w", err)
	}
	if err := r.Client.Create(rc.ctx, job); err != nil {
		if apierrors.IsAlreadyExists(err) {
			rc.Log.Info("skip creating job, it already exists", "job", job.Name)
		} else {
			rc.Log.Error(err, "could not create job", "job", job.Name)
		}
	} else {
		rc.Log.Info("job created", "job", job.Name)
	}
	rc.task.Status.SlicesScheduled = append(rc.task.Status.SlicesScheduled, v1alpha1.ClustercodeSliceRef{
		JobName:    job.Name,
		SliceIndex: index,
	})
	return r.Client.Status().Update(rc.ctx, rc.task)
}

func (r *Reconciler) createMergeJob(rc *ReconciliationContext) error {
	configMountRoot := filepath.Join("/clustercode", controllers.ConfigSubMountPath)
	targetMountRoot := filepath.Join("/clustercode", controllers.TargetSubMountPath)
	variables := map[string]string{
		"${INPUT}":  filepath.Join(configMountRoot, v1alpha1.ConfigMapFileName),
		"${OUTPUT}": filepath.Join(targetMountRoot, rc.task.Spec.TargetUrl.GetPath()),
	}
	job := CreateFfmpegJobDefinition(rc.task, &TaskOpts{
		Args:              MergeArgsAndReplaceVariables(variables, rc.task.Spec.EncodeSpec.DefaultCommandArgs, rc.task.Spec.EncodeSpec.MergeCommandArgs),
		JobType:           controllers.ClustercodeTypeMerge,
		MountIntermediate: true,
		MountTarget:       true,
		MountConfig:       true,
	})
	if err := controllerutil.SetControllerReference(rc.task, job.GetObjectMeta(), r.Scheme); err != nil {
		return fmt.Errorf("could not set controller reference: %w", err)
	}
	if err := r.Client.Create(rc.ctx, job); err != nil {
		if apierrors.IsAlreadyExists(err) {
			rc.Log.Info("skip creating job, it already exists", "job", job.Name)
		} else {
			rc.Log.Error(err, "could not create job", "job", job.Name)
		}
	} else {
		rc.Log.Info("job created", "job", job.Name)
	}
	return nil
}

func getSegmentFileNameTemplatePath(rc *ReconciliationContext, intermediateMountRoot string) string {
	return filepath.Join(intermediateMountRoot, rc.task.Name+"_%d"+filepath.Ext(rc.task.Spec.SourceUrl.GetPath()))
}

func getSourceSegmentFileNameIndexPath(rc *ReconciliationContext, intermediateMountRoot string, index int) string {
	return filepath.Join(intermediateMountRoot, fmt.Sprintf("%s_%d%s", rc.task.Name, index, filepath.Ext(rc.task.Spec.SourceUrl.GetPath())))
}

func getTargetSegmentFileNameIndexPath(rc *ReconciliationContext, intermediateMountRoot string, index int) string {
	return filepath.Join(intermediateMountRoot, fmt.Sprintf("%s_%d%s%s", rc.task.Name, index, v1alpha1.MediaFileDoneSuffix, filepath.Ext(rc.task.Spec.TargetUrl.GetPath())))
}
