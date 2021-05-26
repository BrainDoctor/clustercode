package builder

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
)

func Test_MetaBuilder_WithNamespace(t *testing.T) {
	meta := &metav1.ObjectMeta{
		Namespace: "namespace",
	}
	builder := NewMetaBuilder().WithNamespace("namespace")
	assert.Equal(t, meta, builder.Object)
}

func Test_MetaBuilder_WithName(t *testing.T) {
	meta := &metav1.ObjectMeta{
		Name: "name",
	}
	builder := NewMetaBuilder().WithName("name")
	assert.Equal(t, meta, builder.Object)
}

func Test_MetaBuilder_WithNamespacedName(t *testing.T) {
	meta := &metav1.ObjectMeta{
		Name:      "name",
		Namespace: "namespace",
	}
	builder := NewMetaBuilder().WithNamespacedName(types.NamespacedName{Namespace: "namespace", Name: "name"})
	assert.Equal(t, meta, builder.Object)
}

func Test_MetaBuilder_WithLabels(t *testing.T) {
	meta := &metav1.ObjectMeta{
		Labels: labels.Set{
			"key": "value",
		},
	}
	builder := NewMetaBuilder().WithLabels(labels.Set{
		"key": "value",
	})
	assert.Equal(t, meta, builder.Object)
}

func Test_MetaBuilder_AddLabel(t *testing.T) {
	meta := &metav1.ObjectMeta{
		Labels: labels.Set{
			"key": "value",
		},
	}
	builder := NewMetaBuilderWith(meta).AddLabel("added", "value")
	assert.Equal(t, meta, builder.Object)
}

func Test_MetaBuilder_AddLabels(t *testing.T) {
	meta := &metav1.ObjectMeta{
		Labels: labels.Set{
			"key": "value",
		},
	}
	meta.DeepCopy().Labels["another"] = "value"
	builder := NewMetaBuilderWith(meta).AddLabels(labels.Set{"another": "value"})
	assert.Equal(t, meta, builder.Object)
}
