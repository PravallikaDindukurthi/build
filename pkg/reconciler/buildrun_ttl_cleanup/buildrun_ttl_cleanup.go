// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package buildrun_ttl_cleanup

import (
	"context"
	"fmt"
	"time"

	buildv1alpha1 "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/build/pkg/config"
	"github.com/shipwright-io/build/pkg/ctxlog"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// ReconcileBuildRun reconciles a BuildRun object

type ReconcileBuildRunTtl struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	config                *config.Config
	client                client.Client
	scheme                *runtime.Scheme
	setOwnerReferenceFunc setOwnerReferenceFunc
}

func NewReconciler(c *config.Config, mgr manager.Manager, ownerRef setOwnerReferenceFunc) reconcile.Reconciler {
	return &ReconcileBuildRunTtl{
		config:                c,
		client:                mgr.GetClient(),
		scheme:                mgr.GetScheme(),
		setOwnerReferenceFunc: ownerRef,
	}
}

// GetBuildRunObject retrieves an existing BuildRun based on a name and namespace
func (r *ReconcileBuildRunTtl) GetBuildRunObject(ctx context.Context, objectName string, objectNS string, buildRun *buildv1alpha1.BuildRun) error {
	if err := r.client.Get(ctx, types.NamespacedName{Name: objectName, Namespace: objectNS}, buildRun); err != nil {
		return err
	}
	return nil
}

func DeleteBuildRun(ctx context.Context, rclient client.Client, br *buildv1alpha1.BuildRun, request reconcile.Request) {
	ctxlog.Info(ctx, "Delete build run: ", br.Name, namespace)
	lastTaskRun := &v1beta1.TaskRun{}
	getTaskRunErr := rclient.Get(ctx, types.NamespacedName{Name: *br.Status.LatestTaskRunRef, Namespace: request.Namespace}, lastTaskRun)
	if getTaskRunErr != nil {
		ctxlog.Debug(ctx, "Error getting task run.")
	}
	deleteBuildRunErr := rclient.Delete(ctx, br, &client.DeleteOptions{})
	if deleteBuildRunErr != nil {
		ctxlog.Debug(ctx, "Error deleting buildRun.", br.Name, deleteBuildRunErr)
		fmt.Println(br.Name)
	}

	err := wait.PollImmediate(1*time.Second, 10*time.Second, func() (done bool, err error) {
		buildRun := &buildv1alpha1.BuildRun{}
		err = rclient.Get(ctx, types.NamespacedName{Name: br.Name, Namespace: request.Namespace}, buildRun)
		return apierrors.IsNotFound(err), nil
	})
	if err != nil {
		ctxlog.Debug(ctx, "Error polling for deleting buildrun.")
	}
	err = wait.PollImmediate(1*time.Second, 10*time.Second, func() (done bool, err error) {
		lastTaskRun = &v1beta1.TaskRun{}
		taskRunErr := rclient.Get(ctx, types.NamespacedName{Name: *br.Status.LatestTaskRunRef, Namespace: request.Namespace}, lastTaskRun)
		return apierrors.IsNotFound(taskRunErr), nil
	})
	if err != nil {
		ctxlog.Debug(ctx, "Error deleting the TaskRun..")
	}
	//Return error
}

func (r *ReconcileBuildRunTtl) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	// Set the ctx to be Background, as the top-level context for incoming requests.if
	ctx, cancel := context.WithTimeout(ctx, r.config.CtxTimeOut)
	defer cancel()

	br := &buildv1alpha1.BuildRun{}
	err := r.GetBuildRunObject(ctx, request.Name, request.Namespace, br)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return reconcile.Result{}, err
		}
	}

	if br.Spec.Retention.TtlAfterFailed != nil {
		if br.Status.CompletionTime.Add(br.Spec.Retention.TtlAfterFailed.Duration).After(time.Now()) {
			DeleteBuildRun(ctx, r.client, br, request)
		}
	}

	if br.Spec.Retention.TtlAfterSucceeded != nil {
		if br.Status.CompletionTime.Add(br.Spec.Retention.TtlAfterSucceeded.Duration).After(time.Now()) {
			DeleteBuildRun(ctx, r.client, br, request)
		}
	}

	return reconcile.Result{}, nil
}
