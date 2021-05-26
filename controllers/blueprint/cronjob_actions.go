package blueprint

import (
	"path/filepath"

	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"

	"github.com/ccremer/clustercode/builder"
	"github.com/ccremer/clustercode/cfg"
	"github.com/ccremer/clustercode/controllers"
	"github.com/ccremer/clustercode/controllers/pipeline"
)

func (r *Reconciler) CreateCronJobAction(rc *ReconciliationContext) pipeline.ActionFunc {
	return func() pipeline.Result {
		ctb := builder.NewContainerBuilder("scanner").
			WithImage(cfg.Config.Operator.ClustercodeContainerImage).
			AddEnvVarValue("CC_LOG__DEBUG", "true").
			AddArg("scan").
			AddArg("--namespace=%s", rc.blueprint.Namespace).
			AddArg("--scan.blueprint-name=%s", rc.blueprint.Name)
		psb := builder.NewPodSpecBuilder(ctb).
			AddPvcMount(nil,
				rc.blueprint.Spec.Storage.SourcePvc.ClaimName,
				controllers.SourceSubMountPath,
				filepath.Join("/clustercode", controllers.SourceSubMountPath),
				rc.blueprint.Spec.Storage.SourcePvc.SubPath).
			AddPvcMount(nil,
				rc.blueprint.Spec.Storage.IntermediatePvc.ClaimName,
				controllers.IntermediateSubMountPath,
				filepath.Join("/clustercode", controllers.IntermediateSubMountPath),
				rc.blueprint.Spec.Storage.IntermediatePvc.SubPath).
			Build()

		psb.PodSpec.ServiceAccountName = rc.serviceAccount.Name
		psb.PodSpec.RestartPolicy = corev1.RestartPolicyNever

		cronJob := &v1beta1.CronJob{
			Spec: v1beta1.CronJobSpec{
				Schedule:          rc.blueprint.Spec.ScanSchedule,
				ConcurrencyPolicy: v1beta1.ForbidConcurrent,
				Suspend:           &rc.blueprint.Spec.Suspend,

				JobTemplate: v1beta1.JobTemplateSpec{
					Spec: batchv1.JobSpec{
						BackoffLimit: pointer.Int32Ptr(0),
						Template: corev1.PodTemplateSpec{
							Spec: *psb.PodSpec,
						},
					},
				},
				SuccessfulJobsHistoryLimit: pointer.Int32Ptr(1),
				FailedJobsHistoryLimit:     pointer.Int32Ptr(1),
			},
		}
		builder.NewMetaBuilderWith(cronJob).
			WithLabels(controllers.ClusterCodeLabels, controllers.ClustercodeTypeScan.AsLabels()).
			WithName(rc.blueprint.Name+"-scan-job").
			WithNamespace(rc.blueprint.Namespace).
			WithControllerReference(rc.blueprint, r.Scheme).
			Build()

		if err, op := r.UpsertResource(rc.ctx, cronJob); err != nil {
			r.Recorder.Eventf(rc.blueprint, corev1.EventTypeWarning, "FailedCronJob", "CronJob '%s' could not be created: %v", cronJob.Name, err)
			return pipeline.Result{Err: err, Requeue: true}
		} else if op == pipeline.ResourceCreated {
			r.Recorder.Eventf(rc.blueprint, corev1.EventTypeNormal, "CreatedCronJob", "CronJob '%s' created", cronJob.Name)
		}
		return pipeline.Result{}
	}
}
