// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package build_limit_cleanup

import (
	"context"
	"fmt"
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
type ReconcileBuildLimit struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from :qthe cache and writes to the apiserver
	config                *config.Config
	client                client.Client
	scheme                *runtime.Scheme
	setOwnerReferenceFunc setOwnerReferenceFunc
}

func NewReconciler(c *config.Config, mgr manager.Manager, ownerRef setOwnerReferenceFunc) reconcile.Reconciler {
	return &ReconcileBuildLimit{
		config:                c,
		client:                mgr.GetClient(),
		scheme:                mgr.GetScheme(),
		setOwnerReferenceFunc: ownerRef,
	}
}

func DeleteBuildRun(ctx context.Context, rclient client.Client, br *build.BuildRun, request reconcile.Request) {
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
		buildRun := &build.BuildRun{}
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

func (r *ReconcileBuildLimit) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	// Set the ctx to be Background, as the top-level context for incoming requests.
	ctx, cancel := context.WithTimeout(ctx, r.config.CtxTimeOut)
	defer cancel()

	ctxlog.Debug(ctx, "start reconciling Build-Limit", namespace, request.Namespace, name, request.Name)

	b := &build.Build{}
	err := r.client.Get(ctx, request.NamespacedName, b)
	if err != nil && !apierrors.IsNotFound(err) {
		return reconcile.Result{}, err
	} else if apierrors.IsNotFound(err) {
		ctxlog.Debug(ctx, "finish reconciling build-limit. build was not found", namespace, request.Namespace, name, request.Name)
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

		//Check ttls
		// for _, br := range allBuildRuns.Items {
		// 	if br.Spec.Retention.TtlAfterSucceeded != nil {
		// 		if br.Status.CompletionTime.Add(br.Spec.Retention.TtlAfterSucceeded.Duration).After(time.Now()) {
		// 			DeleteBuildRun(ctx, r.client, &br, request)
		// 		}
		// 	}

		// 	if br.Spec.Retention.TtlAfterFailed != nil {
		// 		if br.Status.CompletionTime.Add(br.Spec.Retention.TtlAfterFailed.Duration).After(time.Now()) {
		// 			DeleteBuildRun(ctx, r.client, &br, request)
		// 		}
		// 	}
		// }

		// allBuildRuns = &build.BuildRunList{}
		// r.client.List(ctx, allBuildRuns, &opts)
		// Check limits
		if b.Spec.Retention.FailedLimit != nil {
			var buildRunFailed []build.BuildRun
			for _, br := range allBuildRuns.Items {
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
				}
			}

			if len(buildRunFailed) > int(*b.Spec.Retention.FailedLimit) {

				// Sort the buildRunFailed based on the Completion time
				sort.Slice(buildRunFailed, func(i, j int) bool {
					return buildRunFailed[i].Status.CompletionTime.Before(buildRunFailed[j].Status.CompletionTime)
				})
				// Delete buildruns
				failedLimit := *b.Spec.Retention.FailedLimit
				lenBuildRun := len(buildRunFailed)
				i := 0
				for lenBuildRun > int(failedLimit) {
					fmt.Println("Deleting BuildRun: --------------------", buildRunFailed[i].Name)
					DeleteBuildRun(ctx, r.client, &buildRunFailed[i], request)
					lenBuildRun -= 1
					i += 1
				}
			}
		}

		if b.Spec.Retention.SucceededLimit != nil {
			var buildRunSucceeded []build.BuildRun
			for _, br := range allBuildRuns.Items {
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
				}
			}
			if len(buildRunSucceeded) > int(*b.Spec.Retention.SucceededLimit) {
				sort.Slice(buildRunSucceeded, func(i, j int) bool {
					return buildRunSucceeded[i].Status.CompletionTime.Before(buildRunSucceeded[j].Status.CompletionTime)
				})

				succeededLimit := *b.Spec.Retention.SucceededLimit
				lenBuildRun := len(buildRunSucceeded)

				i := 0
				for lenBuildRun > int(succeededLimit) {
					fmt.Println("Deleting BuildRun: --------------------", buildRunSucceeded[i].Name)
					DeleteBuildRun(ctx, r.client, &buildRunSucceeded[i], request)
					lenBuildRun -= 1
					i += 1
				}

			}
		}

		// Iterate through all brs, check for ttl, delete if criterion met

	}

	return reconcile.Result{}, nil
}
