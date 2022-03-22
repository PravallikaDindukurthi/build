// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package build_limit_cleanup

import (
	"context"
	"sort"

	build "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/build/pkg/config"
	"github.com/shipwright-io/build/pkg/ctxlog"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
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
<<<<<<< HEAD
<<<<<<< HEAD

		// Check limits
		if b.Spec.Retention.FailedLimit != nil {
			var buildRunFailed []build.BuildRun
			for _, br := range allBuildRuns.Items {
<<<<<<< HEAD
				if br.Status.GetCondition(build.Succeeded).Status == corev1.ConditionFalse {
					buildRunFailed = append(buildRunFailed, br)
<<<<<<< HEAD
<<<<<<< HEAD
<<<<<<< HEAD
					// ctxlog.Debug(ctx, "Failed Build Run. BuildRun name: ", br.Name, namespace)
=======
					ctxlog.Debug(ctx, "failed buildruns list", br)
>>>>>>> cc470c48 (Changes in controller)
=======
					// ctxlog.Debug(ctx, "failed buildruns list", br)
>>>>>>> e0046d8e (Delete buildruns till limit is reached.)
=======
>>>>>>> 1a275132 (Added build-limit-cleanup-controller functionality and added a new controller)
=======

				if br.Status.GetCondition(build.Succeeded) != nil {
					if br.Status.GetCondition(build.Succeeded).Status == corev1.ConditionFalse {
						buildRunFailed = append(buildRunFailed, br)
					}
>>>>>>> eabb7944 (Changes to buildrun.spec. Using buildspec.retention instead.)
=======
=======
		if len(allBuildRuns.Items) == 0 {
			return reconcile.Result{}, nil
		}

>>>>>>> e202a4cb (Made some cleanup and log related changes)
		var buildRunFailed []build.BuildRun
		var buildRunSucceeded []build.BuildRun
		for _, br := range allBuildRuns.Items {
			condition := br.Status.GetCondition(build.Succeeded)
			if condition != nil {
				if condition.Status == corev1.ConditionFalse {
					buildRunFailed = append(buildRunFailed, br)
				} else if condition.Status == corev1.ConditionTrue {
					buildRunSucceeded = append(buildRunSucceeded, br)
>>>>>>> e55875d4 (Watch for buildrun completion in build-limit-cleanup controller.)
				}
			}
		}

		// Check limits
		if b.Spec.Retention.SucceededLimit != nil {
<<<<<<< HEAD
<<<<<<< HEAD
			var buildRunSucceeded []build.BuildRun
			for _, br := range allBuildRuns.Items {
<<<<<<< HEAD
				if br.Status.GetCondition(build.Succeeded).Status == corev1.ConditionTrue {
					buildRunSucceeded = append(buildRunSucceeded, br)
<<<<<<< HEAD
<<<<<<< HEAD
					// ctxlog.Debug(ctx, "Succeeded Build Run. BuildRun name: ", br, namespace)
=======
					ctxlog.Debug(ctx, "succeeded buildruns list", br)
>>>>>>> cc470c48 (Changes in controller)
=======
>>>>>>> 1a275132 (Added build-limit-cleanup-controller functionality and added a new controller)
=======

				if br.Status.GetCondition(build.Succeeded) != nil {
					if br.Status.GetCondition(build.Succeeded).Status == corev1.ConditionTrue {
						buildRunSucceeded = append(buildRunSucceeded, br)
					}
>>>>>>> eabb7944 (Changes to buildrun.spec. Using buildspec.retention instead.)
				}

			}
=======

>>>>>>> e55875d4 (Watch for buildrun completion in build-limit-cleanup controller.)
=======
>>>>>>> e202a4cb (Made some cleanup and log related changes)
			if len(buildRunSucceeded) > int(*b.Spec.Retention.SucceededLimit) {
				sort.Slice(buildRunSucceeded, func(i, j int) bool {
					return buildRunSucceeded[i].Status.CompletionTime.Before(buildRunSucceeded[j].Status.CompletionTime)
				})
				lenOfList := len(buildRunSucceeded)
				for i := 0; lenOfList-i > int(*b.Spec.Retention.SucceededLimit); i += 1 {
					ctxlog.Info(ctx, "Deleting Succeeded buildrun as cleanup limit has been reached.", namespace, request.Namespace, name, buildRunFailed[i].Name)
					br := &buildRunSucceeded[i]
					deleteBuildRunErr := r.client.Delete(ctx, br, &client.DeleteOptions{})
					if deleteBuildRunErr != nil {
						ctxlog.Debug(ctx, "Error deleting buildRun.", br.Name, deleteBuildRunErr)
						return reconcile.Result{}, nil
					}
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
					br := &buildRunFailed[i]
					deleteBuildRunErr := r.client.Delete(ctx, br, &client.DeleteOptions{})
					if deleteBuildRunErr != nil {
						ctxlog.Debug(ctx, "Error deleting buildRun.", br.Name, deleteBuildRunErr)
						return reconcile.Result{}, nil
					}
				}
			}
		}

		ctxlog.Debug(ctx, "finishing reconciling request from a Build or BuildRun event", namespace, request.Namespace, name, request.Name)

	}

	return reconcile.Result{}, nil
}
