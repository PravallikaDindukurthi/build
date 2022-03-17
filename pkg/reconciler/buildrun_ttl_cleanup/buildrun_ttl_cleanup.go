// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package buildrun_ttl_cleanup

import (
	"context"
	"time"

	buildv1alpha1 "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/build/pkg/config"
	"github.com/shipwright-io/build/pkg/ctxlog"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
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

func DeleteBuildRun(ctx context.Context, rclient client.Client, br *buildv1alpha1.BuildRun, request reconcile.Request) error {

	lastTaskRun := &v1beta1.TaskRun{}
	getTaskRunErr := rclient.Get(ctx, types.NamespacedName{Name: *br.Status.LatestTaskRunRef, Namespace: request.Namespace}, lastTaskRun)
	if getTaskRunErr != nil {
		ctxlog.Debug(ctx, "Error getting task run.")
	}

	deleteBuildRunErr := rclient.Delete(ctx, br, &client.DeleteOptions{})
	if deleteBuildRunErr != nil {
		ctxlog.Debug(ctx, "Error deleting buildRun.", br.Name, deleteBuildRunErr)
		return deleteBuildRunErr
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
		taskRunErr := rclient.Get(ctx, types.NamespacedName{Name: lastTaskRun.Name, Namespace: request.Namespace}, lastTaskRun)
		return apierrors.IsNotFound(taskRunErr), nil
	})
	if err != nil {
		ctxlog.Debug(ctx, "Error deleting the TaskRun.")
	}

	return nil
}

func (r *ReconcileBuildRunTtl) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	// Set the ctx to be Background, as the top-level context for incoming requests.if
	ctx, cancel := context.WithTimeout(ctx, r.config.CtxTimeOut)
	defer cancel()

	ctxlog.Debug(ctx, "start reconciling Buildrun-ttl", namespace, request.Namespace, name, request.Name)

	br := &buildv1alpha1.BuildRun{}
	err := r.GetBuildRunObject(ctx, request.Name, request.Namespace, br)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			ctxlog.Debug(ctx, "finish reconciling buildrun-ttl. buildrun was not found", namespace, request.Namespace, name, request.Name)
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, err
	}

	condition := br.Status.GetCondition(buildv1alpha1.Succeeded)
	if condition == nil {
		return reconcile.Result{}, nil
	}

	switch condition.Status {

	case corev1.ConditionTrue:
		if br.Status.BuildSpec.Retention.TtlAfterSucceeded != nil {
			if br.Status.CompletionTime.Add(br.Status.BuildSpec.Retention.TtlAfterSucceeded.Duration).Before(time.Now()) {
				ctxlog.Info(ctx, "Deleting successful buildrun as ttl has been reached.", namespace, request.Namespace, name, request.Name)
				DeleteBuildRun(ctx, r.client, br, request)
			} else {
				timeLeft := br.Status.CompletionTime.Add(br.Status.BuildSpec.Retention.TtlAfterSucceeded.Duration).Sub(time.Now())
				return reconcile.Result{Requeue: true, RequeueAfter: timeLeft}, nil
			}
		}

	case corev1.ConditionFalse:
		if br.Status.BuildSpec.Retention.TtlAfterFailed != nil {
			if br.Status.CompletionTime.Add(br.Status.BuildSpec.Retention.TtlAfterFailed.Duration).Before(time.Now()) {
				ctxlog.Info(ctx, "Deleting failed buildrun as ttl has been reached.", namespace, request.Namespace, name, request.Name)
				DeleteBuildRun(ctx, r.client, br, request)
			} else {
				timeLeft := br.Status.CompletionTime.Add(br.Status.BuildSpec.Retention.TtlAfterFailed.Duration).Sub(time.Now())

				build := &buildv1alpha1.Build{}
				r.client.Get(ctx, types.NamespacedName{Name: *&br.Spec.BuildRef.Name, Namespace: request.Namespace}, build)

				return reconcile.Result{Requeue: true, RequeueAfter: timeLeft}, nil
			}
		}
	}

	return reconcile.Result{}, nil
}
