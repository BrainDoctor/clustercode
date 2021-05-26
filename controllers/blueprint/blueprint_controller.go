package blueprint

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/ccremer/clustercode/api/v1alpha1"
	"github.com/ccremer/clustercode/controllers/pipeline"
)

type (
	// Reconciler reconciles Blueprint objects
	Reconciler struct {
		Recorder record.EventRecorder
		*pipeline.ResourceAction
	}
	// ReconciliationContext holds the parameters of a single reconciliation
	ReconciliationContext struct {
		ctx            context.Context
		blueprint      *v1alpha1.Blueprint
		serviceAccount *corev1.ServiceAccount
		Log            logr.Logger
	}
)

func (r *Reconciler) SetupWithManager(mgr ctrl.Manager, l logr.Logger) error {
	r.ResourceAction = &pipeline.ResourceAction{
		Log:    l,
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}
	r.Recorder = mgr.GetEventRecorderFor("blueprint-controller")
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Blueprint{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(r)
}

// +kubebuilder:rbac:groups=clustercode.github.io,resources=blueprints,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=clustercode.github.io,resources=blueprints/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=batch,resources=cronjobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=batch,resources=cronjobs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core,resources=serviceaccounts,verbs=get;list;create;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles;rolebindings,verbs=get;list;create;delete

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	rc := &ReconciliationContext{
		ctx: ctx,
		blueprint: &v1alpha1.Blueprint{
			ObjectMeta: metav1.ObjectMeta{
				Name:      req.Name,
				Namespace: req.Namespace,
			},
		},
		Log: r.Log.WithValues("blueprint", req.NamespacedName),
	}

	result := pipeline.NewPipeline(rc.Log).
		WithAbortHandler(r.ResourceAction.UpdateStatusHandler(ctx, rc.blueprint)).
		WithSteps(
			pipeline.NewStep("get reconcile object", r.GetOrAbort(rc.ctx, rc.blueprint)),
			pipeline.NewStepWithPredicate("abort if deleted", pipeline.Abort(), pipeline.DeletedPredicate(rc.blueprint)),
			pipeline.NewStep("create service account", r.CreateServiceAccountAction(rc)),
			pipeline.NewStep("create role binding", r.CreateRoleBindingAction(rc)),
			pipeline.NewStep("create cronjob", r.CreateCronJobAction(rc)),
		).Run()
	if result.Err != nil {
		r.Log.Error(result.Err, "pipeline failed with error")
	}
	if result.Requeue {
		return ctrl.Result{RequeueAfter: 1 * time.Minute}, result.Err
	}
	return ctrl.Result{}, result.Err
}
