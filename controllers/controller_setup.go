package controllers

import (
	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
)

// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

type (
	// ReconcilerSetup is a common interface to configure reconcilers.
	ReconcilerSetup interface {
		SetupWithManager(mgr ctrl.Manager, l logr.Logger) error
	}
)
