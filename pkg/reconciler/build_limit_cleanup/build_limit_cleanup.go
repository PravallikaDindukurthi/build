// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package build_limit_cleanup

import (
	"context"
	"sort"
	"time"

	build "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/build/pkg/config"
	"github.com/shipwright-io/build/pkg/ctxlog"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// ReconcileBuild reconciles a Build object
type ReconcileBuild struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from :qthe cache and writes to the apiserver
	config                *config.Config
	client                client.Client
	scheme                *runtime.Scheme
	setOwnerReferenceFunc setOwnerReferenceFunc
}

func NewReconciler(c *config.Config, mgr manager.Manager, ownerRef setOwnerReferenceFunc) reconcile.Reconciler {
	return &ReconcileBuild{
		config:                c,
		client:                mgr.GetClient(),
		scheme:                mgr.GetScheme(),
		setOwnerReferenceFunc: ownerRef,
	}
}

func DeleteBuildRun(ctx context.Context, rclient client.Client, br *build.BuildRun, request reconcile.Request) error {

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
		buildRun := &build.BuildRun{}
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

func (r *ReconcileBuild) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	// Set the ctx to be Background, as the top-level context for incoming requests.
	ctx, cancel := context.WithTimeout(ctx, r.config.CtxTimeOut)
	defer cancel()

	ctxlog.Debug(ctx, "start reconciling build-limit-cleanup", namespace, request.Namespace, name, request.Name)

	b := &build.Build{}
	err := r.client.Get(ctx, request.NamespacedName, b)
	if err != nil && !apierrors.IsNotFound(err) {
		return reconcile.Result{}, err
	} else if apierrors.IsNotFound(err) {
		ctxlog.Debug(ctx, "finish reconciling build-limit-cleanup. Build was not found", namespace, request.Namespace, name, request.Name)
		return reconcile.Result{}, nil
	}

	// Check if retention is set. If so, get all corresponding BR, else, return.
	// TTL deletions happen regardless of whether retention fields are set or not
	if b.Spec.Retention != nil {

		lbls := map[string]string{
			build.LabelBuild: b.Name,
		}
		opts := client.ListOptions{
			Namespace:     b.Namespace,
			LabelSelector: labels.SelectorFromSet(lbls),
		}
		allBuildRuns := &build.BuildRunList{}
		r.client.List(ctx, allBuildRuns, &opts)
		if len(allBuildRuns.Items) == 0 {
			return reconcile.Result{}, nil
		}

		var buildRunFailed []build.BuildRun
		var buildRunSucceeded []build.BuildRun
		for _, br := range allBuildRuns.Items {
			condition := br.Status.GetCondition(build.Succeeded)
			if condition != nil {
				if condition.Status == corev1.ConditionFalse {
					buildRunFailed = append(buildRunFailed, br)
				} else if condition.Status == corev1.ConditionTrue {
					buildRunSucceeded = append(buildRunSucceeded, br)
				}
			}
		}

		// Check limits
		if b.Spec.Retention.SucceededLimit != nil {
			if len(buildRunSucceeded) > int(*b.Spec.Retention.SucceededLimit) {
				sort.Slice(buildRunSucceeded, func(i, j int) bool {
					return buildRunSucceeded[i].Status.CompletionTime.Before(buildRunSucceeded[j].Status.CompletionTime)
				})
				lenOfList := len(buildRunSucceeded)
				for i := 0; lenOfList-i > int(*b.Spec.Retention.SucceededLimit); i += 1 {
					ctxlog.Info(ctx, "Deleting Succeeded buildrun as cleanup limit has been reached.", namespace, request.Namespace, name, buildRunFailed[i].Name)
					DeleteBuildRun(ctx, r.client, &buildRunSucceeded[i], request)
				}
			}
		}

		if b.Spec.Retention.FailedLimit != nil {
			if len(buildRunFailed) > int(*b.Spec.Retention.FailedLimit) {
				sort.Slice(buildRunFailed, func(i, j int) bool {
					return buildRunFailed[i].Status.CompletionTime.Before(buildRunFailed[j].Status.CompletionTime)
				})
				lenOfList := len(buildRunFailed)
				for i := 0; lenOfList-i > int(*b.Spec.Retention.FailedLimit); i += 1 {
					ctxlog.Info(ctx, "Deleting failed buildrun as cleanup limit has been reached.", namespace, request.Namespace, name, buildRunFailed[i].Name)
					DeleteBuildRun(ctx, r.client, &buildRunFailed[i], request)
				}
			}
		}

		ctxlog.Debug(ctx, "finishing reconciling request from a Build or BuildRun event", namespace, request.Namespace, name, request.Name)

	}

	return reconcile.Result{}, nil
}
