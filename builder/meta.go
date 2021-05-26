package builder

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type (
	MetaProperty interface {
		Apply(b *MetaBuilder)
	}

	MetaBuilder struct {
		Object metav1.Object
	}
)

func NewMetaBuilder() *MetaBuilder {
	return NewMetaBuilderWith(&metav1.ObjectMeta{})
}

func NewMetaBuilderWith(obj metav1.Object) *MetaBuilder {
	return &MetaBuilder{Object: obj}
}

func (b MetaBuilder) Build(props ...MetaProperty) MetaBuilder {
	for _, opt := range props {
		opt.Apply(&b)
	}
	return b
}

func (b *MetaBuilder) WithNamespace(ns string) *MetaBuilder {
	b.Object.SetNamespace(ns)
	return b
}

func (b *MetaBuilder) WithName(name string) *MetaBuilder {
	b.Object.SetName(name)
	return b
}

func (b *MetaBuilder) WithNamespacedName(nsd types.NamespacedName) *MetaBuilder {
	b.Object.SetName(nsd.Name)
	b.Object.SetNamespace(nsd.Namespace)
	return b
}

func (b *MetaBuilder) WithControllerReference(owner client.Object, scheme *runtime.Scheme) *MetaBuilder {
	_ = controllerutil.SetControllerReference(owner, b.Object, scheme)
	return b
}

func (b *MetaBuilder) WithLabels(sets ...labels.Set) *MetaBuilder {
	s := labels.Set{}
	for _, set := range sets {
		s = labels.Merge(s, set)
	}
	b.Object.SetLabels(s)
	return b
}

func (b *MetaBuilder) AddLabel(key, value string) *MetaBuilder {
	l := b.Object.GetLabels()
	if l == nil {
		l = labels.Set{}
	}
	l[key] = value
	b.Object.SetLabels(l)
	return b
}

func (b *MetaBuilder) AddLabels(set labels.Set) *MetaBuilder {
	l := b.Object.GetLabels()
	if l == nil {
		l = labels.Set{}
	}
	b.Object.SetLabels(labels.Merge(l, set))
	return b
}
