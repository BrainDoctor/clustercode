package builder

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
)

type (
	ContainerProperty interface {
		Apply(b *ContainerBuilder)
	}

	ContainerBuilder struct {
		Container *corev1.Container
	}
)

func NewContainerBuilder(containerName string) *ContainerBuilder {
	return NewContainerBuilderWith(&corev1.Container{Name: containerName})
}

func NewContainerBuilderWith(container *corev1.Container) *ContainerBuilder {
	return &ContainerBuilder{Container: container}
}

func (b *ContainerBuilder) Build(props ...ContainerProperty) *ContainerBuilder {
	for _, prop := range props {
		prop.Apply(b)
	}
	return b
}

func (b *ContainerBuilder) WithName(name string) *ContainerBuilder {
	b.Container.Name = name
	return b
}

func (b *ContainerBuilder) WithImage(image string) *ContainerBuilder {
	b.Container.Image = image
	return b
}

func (b *ContainerBuilder) WithImagePullPolicy(policy corev1.PullPolicy) *ContainerBuilder {
	b.Container.ImagePullPolicy = policy
	return b
}

func (b *ContainerBuilder) WithArgs(args ...string) *ContainerBuilder {
	b.Container.Args = args
	return b
}

func (b *ContainerBuilder) AddArg(format string, args ...interface{}) *ContainerBuilder {
	b.Container.Args = append(b.Container.Args, fmt.Sprintf(format, args...))
	return b
}

func (b *ContainerBuilder) AddArgs(args ...string) *ContainerBuilder {
	b.Container.Args = append(b.Container.Args, args...)
	return b
}

func (b *ContainerBuilder) AddEnvVarValue(key, format string, args ...interface{}) *ContainerBuilder {
	b.Container.Env = append(b.Container.Env, corev1.EnvVar{Name: key, Value: fmt.Sprintf(format, args...)})
	return b
}

func (b *ContainerBuilder) AddEnvFromConfigMap(configMapName, prefix string) *ContainerBuilder {
	b.Container.EnvFrom = append(b.Container.EnvFrom, corev1.EnvFromSource{
		Prefix: prefix,
		ConfigMapRef: &corev1.ConfigMapEnvSource{
			LocalObjectReference: corev1.LocalObjectReference{Name: configMapName},
		},
	})
	return b
}

func (b *ContainerBuilder) AddEnvValueFromConfigMap(configMapName, key string) *ContainerBuilder {
	b.Container.Env = append(b.Container.Env, corev1.EnvVar{
		Name: key,
		ValueFrom: &corev1.EnvVarSource{
			ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{Name: configMapName},
				Key:                  key,
			},
		},
	})
	return b
}

func (b *ContainerBuilder) AddVolumeMount(mount corev1.VolumeMount) *ContainerBuilder {
	b.Container.VolumeMounts = append(b.Container.VolumeMounts, mount)
	return b
}

func (b *ContainerBuilder) AddMountPath(volumeName, mountPath, subPath string) *ContainerBuilder {
	b.AddVolumeMount(corev1.VolumeMount{
		Name:      volumeName,
		MountPath: mountPath,
		SubPath:   subPath,
	})
	return b
}
