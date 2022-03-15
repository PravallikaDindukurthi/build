// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package build_limit_cleanup

import (
	"context"

	build "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
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

// Add creates a new Build Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(ctx context.Context, c *config.Config, mgr manager.Manager) error {
	ctx = ctxlog.NewContext(ctx, "build-limit-cleanup-controller")
	return add(ctx, mgr, NewReconciler(c, mgr, controllerutil.SetControllerReference), c.Controllers.Build.MaxConcurrentReconciles)
}

func add(ctx context.Context, mgr manager.Manager, r reconcile.Reconciler, maxConcurrentReconciles int) error {
	// Create the controller options
	options := controller.Options{
		Reconciler: r,
	}

	if maxConcurrentReconciles > 0 {
		options.MaxConcurrentReconciles = maxConcurrentReconciles
	}

	// Create a new controller
	c, err := controller.New("build-limit-cleanup-controller", mgr, options)
	if err != nil {
		return err
	}

	pred := predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			o := e.Object.(*build.Build)
			if o.Spec.Retention != nil {
				return true
			}
			return false
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			n := e.ObjectNew.(*build.Build)
			// Check if the Retention field exists
			if n.Spec.Retention != nil {
				return true
			}
			return false
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			// Never reconcile on deletion, there is nothing we have to do
			return false
		},
	}

	// Watch for changes to primary resource Build
	if err = c.Watch(&source.Kind{Type: &build.Build{}}, &handler.EnqueueRequestForObject{}, pred); err != nil {
		return err
	}

	return nil
}
