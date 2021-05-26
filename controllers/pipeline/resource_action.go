package pipeline

import (
	"context"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ResourceAction struct {
	Log    logr.Logger
	Client client.Client
	Scheme *runtime.Scheme
}

type ResourceOperation string

const (
	ResourceCreated         = "Created"
	ResourceUpdated         = "Updated"
	ResourceNoop            = "Noop"
	ResourceOperationFailed = "OperationFailed"
)

func (r *ResourceAction) GetOrAbort(ctx context.Context, obj client.Object) ActionFunc {
	return func() Result {
		name := MapToNamespacedName(obj)
		err := r.Client.Get(ctx, name, obj)
		if err != nil {
			if apierrors.IsNotFound(err) {
				r.Log.Info("object not found", "object", name.String())
				return Result{Abort: true}
			}
			return Result{Err: err, Requeue: true}
		}
		r.Log.Info("Reconciling object")
		return Result{}
	}
}

// CreateIfNotExisting creates the resource if it doesn't exist.
// It returns true if the resource existed before.
func (r *ResourceAction) CreateIfNotExisting(ctx context.Context, obj client.Object) (error, ResourceOperation) {
	name := MapToNamespacedName(obj)
	if err := r.Client.Create(ctx, obj); err != nil {
		if apierrors.IsAlreadyExists(err) {
			r.Log.V(1).Info("object already exists", "object", name.String())
			return nil, ResourceNoop
		}
		return err, ResourceOperationFailed
	}
	r.Log.Info("created object", "object", name.String(), "kind", obj.GetObjectKind().GroupVersionKind().Kind)
	return nil, ResourceCreated
}

func (r *ResourceAction) UpsertResource(ctx context.Context, object client.Object) (error, ResourceOperation) {
	name := MapToNamespacedName(object)
	if updateErr := r.Client.Update(ctx, object); updateErr != nil {
		if apierrors.IsNotFound(updateErr) {
			if createErr := r.Client.Create(ctx, object); createErr != nil {
				r.Log.Error(createErr, "could not create resource", "resource", name.String())
				return createErr, ResourceOperationFailed
			}
			r.Log.V(1).Info("resource created", "resource", name.String())
			return nil, ResourceCreated
		}
		r.Log.Error(updateErr, "could not update resource", "resource", name.String())
		return updateErr, ResourceOperationFailed
	}
	r.Log.V(1).Info("resource updated", "resource", name.String())
	return nil, ResourceUpdated
}

func (r *ResourceAction) UpdateStatusHandler(ctx context.Context, obj client.Object) Handler {
	return func(result Result) {
		if err := r.Client.Status().Update(ctx, obj); err != nil {
			r.Log.Error(err, "could not update status")
		}
		r.Log.V(1).Info("updated status")
	}
}
