// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package build_limit_cleanup

import (
	"context"
	"fmt"
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

	// Check if limit is set. If so, get all corresponding BR, else, return.
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
		var buildRunSucceeded []build.BuildRun
		var buildRunFailed []build.BuildRun
		if b.Spec.Retention.FailedLimit != nil {
			for _, br := range allBuildRuns.Items {
				if br.Status.GetCondition(build.Succeeded).Status == corev1.ConditionFalse {
					buildRunFailed = append(buildRunFailed, br)
<<<<<<< HEAD
<<<<<<< HEAD
					// ctxlog.Debug(ctx, "Failed Build Run. BuildRun name: ", br.Name, namespace)
=======
					ctxlog.Debug(ctx, "failed buildruns list", br)
>>>>>>> cc470c48 (Changes in controller)
=======
					// ctxlog.Debug(ctx, "failed buildruns list", br)
>>>>>>> e0046d8e (Delete buildruns till limit is reached.)
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
					// lastTaskRun := &v1beta1.TaskRun{}
					//getTaskRunErr := r.client.Get(ctx, types.NamespacedName{Name: buildRunFailed[i].Name, Namespace: request.Namespace}, lastTaskRun)
					deleteBuildRunErr := r.client.Delete(ctx, &buildRunFailed[i], &client.DeleteOptions{})
					if deleteBuildRunErr != nil {
						ctxlog.Debug(ctx, "Error deleting buildRun - ", buildRunFailed[i].Name, err)
					}
					fmt.Println("Deleted BuildRun: ", buildRunFailed[i].Name)
					// ctxlog.Debug(ctx, "Get task run result: ", lastTaskRun.Name, namespace)
					lenBuildRun -= 1
					i += 1
					// get task run
					// delete the task run
					// delete build run
					// poll to see if they are deleted <-?

				}
			}
		}

		if b.Spec.Retention.SucceededLimit != nil {
			for _, br := range allBuildRuns.Items {
				if br.Status.GetCondition(build.Succeeded).Status == corev1.ConditionTrue {
					buildRunSucceeded = append(buildRunSucceeded, br)
<<<<<<< HEAD
					// ctxlog.Debug(ctx, "Succeeded Build Run. BuildRun name: ", br, namespace)
=======
					ctxlog.Debug(ctx, "succeeded buildruns list", br)
>>>>>>> cc470c48 (Changes in controller)
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
					// lastTaskRun := &v1beta1.TaskRun{}
					// getTaskRunErr := r.client.Get(ctx, types.NamespacedName{Name: *buildRunSucceeded[i].Status.LatestTaskRunRef, Namespace: request.Namespace}, lastTaskRun)
					// getTaskRunErr := buildRunSucceeded[i].Status.LatestTaskRunRef
					deleteBuildRunErr := r.client.Delete(ctx, &buildRunSucceeded[i], &client.DeleteOptions{})
					if deleteBuildRunErr != nil {
						// ctxlog.Debug(ctx, "Error getting task run for - ", buildRunSucceeded[i], err)
						ctxlog.Debug(ctx, "Error deleting buildRun - ", buildRunFailed[i].Name, err)
						fmt.Println(buildRunSucceeded[i].Name)
					}

					// ctxlog.Debug(ctx, "Get task run result: ", lastTaskRun.Name, namespace)
					fmt.Println("Deleted BuildRun: ", buildRunFailed[i].Name)

					// 	// get task run
					// 	// delete the task run
					// 	// delete build run
					// 	// poll to see if they are deleted <-?
					i += 1
					lenBuildRun -= 1

				}
				fmt.Println("Task run end: -----")

			}
			// Delete Buildruns
		}

	}

	return reconcile.Result{}, nil
}
