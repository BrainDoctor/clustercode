package builder

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func Test_ContainerBuilder_WithName(t *testing.T) {
	container := &corev1.Container{
		Name: "name",
	}
	builder := NewContainerBuilder("name")
	assert.Equal(t, container, builder.Container)
}

func Test_ContainerBuilder_WithImage(t *testing.T) {
	container := &corev1.Container{
		Name:  "name",
		Image: "image",
	}
	builder := NewContainerBuilder("name").WithImage("image")
	assert.Equal(t, container, builder.Container)
}

func Test_ContainerBuilder_WithImagePullPolicy(t *testing.T) {
	container := &corev1.Container{
		Name:            "name",
		ImagePullPolicy: corev1.PullIfNotPresent,
	}
	builder := NewContainerBuilder("name").WithImagePullPolicy(corev1.PullIfNotPresent)
	assert.Equal(t, container, builder.Container)
}

func Test_ContainerBuilder_WithArgs(t *testing.T) {
	container := &corev1.Container{
		Name: "name",
		Args: []string{"args1", "arg2"},
	}
	builder := NewContainerBuilder("name").WithArgs("args1", "arg2")
	assert.Equal(t, container, builder.Container)
}

func Test_ContainerBuilder_AddArg(t *testing.T) {
	container := &corev1.Container{
		Name: "name",
		Args: []string{fmt.Sprintf("arg%d", 2)},
	}
	builder := NewContainerBuilder("name").AddArg("arg%d", 2)
	assert.Equal(t, container, builder.Container)
}

func Test_ContainerBuilder_AddArgs(t *testing.T) {
	container := &corev1.Container{
		Name: "name",
		Args: []string{"arg1"},
	}
	container.Args = append(container.Args, "arg2")
	builder := NewContainerBuilder("name").WithArgs("arg1").AddArgs("arg2")
	assert.Equal(t, container, builder.Container)
}

func Test_ContainerBuilder_AddEnvVarValue(t *testing.T) {
	container := &corev1.Container{
		Name: "name",
		Env: []corev1.EnvVar{
			{
				Name:  "key",
				Value: "value",
			},
		},
	}
	builder := NewContainerBuilder("name").AddEnvVarValue("key", "value")
	assert.Equal(t, container, builder.Container)
}

func Test_ContainerBuilder_AddEnvValueFromConfigMap(t *testing.T) {
	container := &corev1.Container{
		Name: "name",
		Env: []corev1.EnvVar{
			{
				Name: "key",
				ValueFrom: &corev1.EnvVarSource{
					ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{Name: "configmap"},
						Key:                  "key",
					}},
			},
		},
	}
	builder := NewContainerBuilder("name").AddEnvValueFromConfigMap("configmap", "key")
	assert.Equal(t, container, builder.Container)
}

func Test_ContainerBuilder_AddEnvFromConfigMap(t *testing.T) {
	container := &corev1.Container{
		Name: "name",
		EnvFrom: []corev1.EnvFromSource{
			{
				Prefix: "prefix",
				ConfigMapRef: &corev1.ConfigMapEnvSource{
					LocalObjectReference: corev1.LocalObjectReference{Name: "configmap"},
				},
			},
		},
	}
	builder := NewContainerBuilder("name").AddEnvFromConfigMap("configmap", "prefix")
	assert.Equal(t, container, builder.Container)
}

func Test_ContainerBuilder_AddMountPath(t *testing.T) {
	container := &corev1.Container{
		Name: "name",
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "volume",
				MountPath: "mountPath",
				SubPath:   "subPath",
			},
		},
	}
	builder := NewContainerBuilder("name").AddMountPath("volume", "mountPath", "subPath")
	assert.Equal(t, container, builder.Container)
}
