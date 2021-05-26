package builder

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func Test_PodSpecBuilder_AddPvcMount(t *testing.T) {
	podSpec := &corev1.PodSpec{
		Containers: []corev1.Container{
			{
				Name: "container",
				VolumeMounts: []corev1.VolumeMount{
					{
						Name:      "volume",
						MountPath: "mountPath",
						SubPath:   "subPath",
					},
				},
			},
		},
		Volumes: []corev1.Volume{
			{
				Name: "volume",
				VolumeSource: corev1.VolumeSource{
					PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
						ClaimName: "claim",
					},
				},
			},
		},
	}
	builder := NewPodSpecBuilder(NewContainerBuilder("container")).
		AddPvcMount(nil, "claim", "volume", "mountPath", "subPath").
		Build()

	assert.Equal(t, podSpec, builder.PodSpec)
}

func Test_PodSpecBuilder_AddConfigMapMount(t *testing.T) {
	podSpec := &corev1.PodSpec{
		Volumes: []corev1.Volume{
			{
				Name: "volume",
				VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{Name: "configmap"},
				}},
			},
		},
		Containers: []corev1.Container{
			{
				Name: "container",
				VolumeMounts: []corev1.VolumeMount{
					{
						Name:      "volume",
						MountPath: "mountPath",
					},
				},
			},
		},
	}

	builder := NewPodSpecBuilder(NewContainerBuilder("container")).
		AddConfigMapMount(nil, "configmap", "volume", "mountPath").
		Build()

	assert.Equal(t, podSpec, builder.PodSpec)
}
