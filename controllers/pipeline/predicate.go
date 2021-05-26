package pipeline

import "sigs.k8s.io/controller-runtime/pkg/client"

func DeletedPredicate(obj client.Object) Predicate {
	return func(step Step) bool {
		return obj.GetDeletionTimestamp() != nil
	}
}

func TruePredicate() Predicate {
	return func(step Step) bool {
		return true
	}
}
