// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package buildrun_ttl_cleanup

import (
	"context"

	buildv1alpha1 "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/build/pkg/config"
	"github.com/shipwright-io/build/pkg/ctxlog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	namespace string = "namespace"
	name      string = "name"
)

type setOwnerReferenceFunc func(owner, object metav1.Object, scheme *runtime.Scheme) error

// Add creates a new BuildRun Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(ctx context.Context, c *config.Config, mgr manager.Manager) error {
	ctx = ctxlog.NewContext(ctx, "buildrun-ttl-cleanup-controller")
	return add(ctx, mgr, NewReconciler(c, mgr, controllerutil.SetControllerReference), c.Controllers.BuildRun.MaxConcurrentReconciles)
}

func add(ctx context.Context, mgr manager.Manager, r reconcile.Reconciler, maxConcurrentReconciles int) error {
	// Create the controller options
	options := controller.Options{
		Reconciler: r,
	}

	if maxConcurrentReconciles > 0 {
		options.MaxConcurrentReconciles = maxConcurrentReconciles

	}

	c, err := controller.New("buildrun-ttl-cleanup-controller", mgr, options)
	if err != nil {
		return err
	}

	ctxlog.Debug(ctx, "start reconciling BuildRun-ttl.", namespace)

	predBuildRun := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			// Check if retention -> ttl exists
			return true
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			// Never reconcile on deletion, there is nothing we have to do
			return false
		},
	}
	// Watch for changes to primary resource BuildRun // Requeue after some time
	if err = c.Watch(&source.Kind{Type: &buildv1alpha1.BuildRun{}}, &handler.EnqueueRequestForObject{}, predBuildRun); err != nil {
		return err
	}
	return nil
}
