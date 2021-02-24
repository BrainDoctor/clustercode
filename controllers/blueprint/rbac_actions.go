package blueprint

import (
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/utils/strings"

	"github.com/ccremer/clustercode/builder"
	"github.com/ccremer/clustercode/cfg"
	"github.com/ccremer/clustercode/controllers"
	"github.com/ccremer/clustercode/controllers/pipeline"
)

func (r *Reconciler) CreateServiceAccountAction(rc *ReconciliationContext) pipeline.ActionFunc {
	return func() pipeline.Result {
		rc.serviceAccount = r.newServiceAccount(rc)
		err, op := r.CreateIfNotExisting(rc.ctx, rc.serviceAccount)
		if err != nil {
			r.Recorder.Eventf(rc.blueprint, corev1.EventTypeWarning, "FailedServiceAccountCreation", "ServiceAccount '%' could not be created: %v", rc.serviceAccount.Name, err)
		} else if op == pipeline.ResourceCreated {
			r.Recorder.Eventf(rc.blueprint, corev1.EventTypeNormal, "ServiceAccountCreated", "ServiceAccount '%s' created", rc.serviceAccount.Name)
		}
		return pipeline.Result{Err: err}
	}
}

func (r *Reconciler) CreateRoleBindingAction(rc *ReconciliationContext) pipeline.ActionFunc {
	return func() pipeline.Result {
		binding := r.newRoleBinding(rc)
		err, op := r.CreateIfNotExisting(rc.ctx, binding)
		if err != nil {
			r.Recorder.Eventf(rc.blueprint, corev1.EventTypeWarning, "FailedRoleBindingCreation", "RoleBinding '%' could not be created: %v", binding.Name, err)
		} else if op == pipeline.ResourceCreated {
			r.Recorder.Eventf(rc.blueprint, corev1.EventTypeNormal, "RoleBindingCreated", "RoleBinding '%s' created", binding.Name)
		}
		return pipeline.Result{Err: err}
	}
}

func (r *Reconciler) newServiceAccount(rc *ReconciliationContext) *corev1.ServiceAccount {
	saName := rc.blueprint.GetServiceAccountName()
	account := &corev1.ServiceAccount{}
	builder.NewMetaBuilderWith(account).
		WithName(saName).
		WithNamespace(rc.blueprint.Namespace).
		WithLabels(controllers.ClusterCodeLabels).
		WithControllerReference(rc.blueprint, r.Scheme)
	return account
}

func (r *Reconciler) newRoleBinding(rc *ReconciliationContext) *rbacv1.RoleBinding {
	saName := rc.blueprint.GetServiceAccountName()
	roleBinding := &rbacv1.RoleBinding{
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Namespace: rc.blueprint.Namespace,
				Name:      saName,
			},
		},
		RoleRef: rbacv1.RoleRef{
			Kind:     cfg.Config.Scan.RoleKind,
			Name:     cfg.Config.Scan.RoleName,
			APIGroup: rbacv1.GroupName,
		},
	}
	builder.NewMetaBuilderWith(roleBinding).
		WithName(strings.ShortenString(saName, 51)+"-rolebinding").
		WithNamespace(rc.blueprint.Namespace).
		WithLabels(controllers.ClusterCodeLabels).
		WithControllerReference(rc.blueprint, r.ResourceAction.Scheme)
	return roleBinding
}
