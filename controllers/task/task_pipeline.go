package task

import (
	"fmt"
	"path/filepath"
	"strings"

	"k8s.io/api/batch/v1"
	v12 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"

	"github.com/ccremer/clustercode/api/v1alpha1"
	"github.com/ccremer/clustercode/builder"
	"github.com/ccremer/clustercode/cfg"
	"github.com/ccremer/clustercode/controllers"
	"github.com/ccremer/clustercode/controllers/pipeline"
)

func MergeArgsAndReplaceVariables(variables map[string]string, argsList ...[]string) (merged []string) {
	for _, args := range argsList {
		for _, arg := range args {
			for k, v := range variables {
				arg = strings.ReplaceAll(arg, k, v)
			}
			merged = append(merged, arg)
		}
	}
	return merged
}

func splitJobPredicate(rc *ReconciliationContext) pipeline.Predicate {
	return func(step pipeline.Step) bool {
		return rc.task.Spec.SlicesPlannedCount == 0
	}
}

func mergeJobPredicate(rc *ReconciliationContext) pipeline.Predicate {
	return func(step pipeline.Step) bool {
		return len(rc.task.Status.SlicesFinished) >= rc.task.Spec.SlicesPlannedCount
	}
}

func CreateFfmpegJobDefinition(task *v1alpha1.Task, opts *TaskOpts) *v1.Job {
	cb := builder.NewContainerBuilder("ffmpeg").
		WithImage(cfg.Config.Operator.FfmpegContainerImage).
		WithImagePullPolicy(v12.PullIfNotPresent).
		WithArgs(opts.Args...).
		Build()

	pb := builder.NewPodSpecBuilder(cb).Build()
	pb.PodSpec.ServiceAccountName = task.Spec.ServiceAccountName
	pb.PodSpec.SecurityContext = &v12.PodSecurityContext{
		RunAsUser:  pointer.Int64Ptr(1000),
		RunAsGroup: pointer.Int64Ptr(0),
		FSGroup:    pointer.Int64Ptr(0),
	}
	pb.PodSpec.RestartPolicy = v12.RestartPolicyNever

	if opts.MountSource {
		pvc := task.Spec.Storage.SourcePvc
		pb.AddPvcMount(nil, controllers.SourceSubMountPath, pvc.ClaimName, filepath.Join("/clustercode", controllers.SourceSubMountPath), pvc.SubPath)
	}
	if opts.MountIntermediate {
		pvc := task.Spec.Storage.IntermediatePvc
		pb.AddPvcMount(nil, controllers.IntermediateSubMountPath, pvc.ClaimName, filepath.Join("/clustercode", controllers.IntermediateSubMountPath), pvc.SubPath)
	}
	if opts.MountTarget {
		pvc := task.Spec.Storage.TargetPvc
		pb.AddPvcMount(nil, controllers.TargetSubMountPath, pvc.ClaimName, filepath.Join("/clustercode", controllers.TargetSubMountPath), pvc.SubPath)
	}
	if opts.MountConfig {
		pb.AddConfigMapMount(nil, controllers.ConfigSubMountPath, task.Spec.FileListConfigMapRef, filepath.Join("/clustercode", controllers.ConfigSubMountPath))
	}

	job := &v1.Job{
		Spec: v1.JobSpec{
			BackoffLimit: pointer.Int32Ptr(0),
			Template: v12.PodTemplateSpec{
				Spec: *pb.Build().PodSpec,
			},
		},
	}
	builder.NewMetaBuilderWith(job).
		WithNamespace(task.Namespace).
		WithName(fmt.Sprintf("%s-%s", task.Spec.TaskId, opts.JobType)).
		WithLabels(controllers.ClusterCodeLabels, opts.JobType.AsLabels(), task.Spec.TaskId.AsLabels()).
		Build()
	return job
}
