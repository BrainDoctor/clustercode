package controllers

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
)

type (
	ClusterCodeJobType string
)

var (
	ClusterCodeLabels = labels.Set{
		"app.kubernetes.io/managed-by": "clustercode",
	}
)

const (
	SourceSubMountPath       = "source"
	TargetSubMountPath       = "target"
	IntermediateSubMountPath = "intermediate"
	ConfigSubMountPath       = "config"

	ClustercodeTypeLabelKey                          = "clustercode.github.io/type"
	ClustercodeSliceIndexLabelKey                    = "clustercode.github.io/slice-index"
	ClustercodeTypeScan           ClusterCodeJobType = "scan"
	ClustercodeTypeSplit          ClusterCodeJobType = "split"
	ClustercodeTypeSlice          ClusterCodeJobType = "slice"
	ClustercodeTypeCount          ClusterCodeJobType = "count"
	ClustercodeTypeMerge          ClusterCodeJobType = "merge"
	ClustercodeTypeCleanup        ClusterCodeJobType = "cleanup"
)

var (
	ClustercodeTypes = []ClusterCodeJobType{
		ClustercodeTypeScan, ClustercodeTypeSplit, ClustercodeTypeCount, ClustercodeTypeSlice,
		ClustercodeTypeMerge, ClustercodeTypeCleanup}
)

func (t ClusterCodeJobType) AsLabels() labels.Set {
	return labels.Set{
		ClustercodeTypeLabelKey: string(t),
	}
}

func (t ClusterCodeJobType) String() string {
	return string(t)
}

func getOwner(obj metav1.Object) types.NamespacedName {
	for _, owner := range obj.GetOwnerReferences() {
		if pointer.BoolPtrDerefOr(owner.Controller, false) {
			return types.NamespacedName{Namespace: obj.GetNamespace(), Name: owner.Name}
		}
	}
	return types.NamespacedName{}
}
