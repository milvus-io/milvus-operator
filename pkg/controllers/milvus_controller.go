/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"
	"time"

	"helm.sh/helm/v3/pkg/cli"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/go-logr/logr"
	milvusv1alpha1 "github.com/milvus-io/milvus-operator/apis/milvus.io/v1alpha1"
	"github.com/milvus-io/milvus-operator/pkg/config"
)

const (
	MilvusFinalizerName = "milvus.milvus.io/finalizer"
)

// MilvusReconciler reconciles a Milvus object
type MilvusReconciler struct {
	client.Client
	Scheme         *runtime.Scheme
	logger         logr.Logger
	helmSettings   *cli.EnvSettings
	helmReconciler HelmReconciler
	statusSyncer   *MilvusStatusSyncer
}

//+kubebuilder:rbac:groups=milvus.io,resources=milvus,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=milvus.io,resources=milvus/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=milvus.io,resources=milvus/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Milvus object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.10.0/pkg/reconcile
func (r *MilvusReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.statusSyncer.RunIfNot()
	if !config.IsDebug() {
		defer func() {
			if err := recover(); err != nil {
				r.logger.Error(err.(error), "reconcile panic")
			}
		}()
	}

	milvus := &milvusv1alpha1.Milvus{}
	if err := r.Get(ctx, req.NamespacedName, milvus); err != nil {
		if errors.IsNotFound(err) {
			// The resource may have be deleted after reconcile request coming in
			// Reconcile is done
			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, fmt.Errorf("error get milvus cluster: %w", err)
	}

	// Finalize
	if milvus.ObjectMeta.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(milvus, MilvusFinalizerName) {
			controllerutil.AddFinalizer(milvus, MilvusFinalizerName)
			err := r.Update(ctx, milvus)
			return ctrl.Result{}, err
		}

	} else {
		if controllerutil.ContainsFinalizer(milvus, MilvusFinalizerName) {
			if err := r.Finalize(ctx, *milvus); err != nil {
				return ctrl.Result{}, err
			}
			controllerutil.RemoveFinalizer(milvus, MilvusFinalizerName)
			err := r.Update(ctx, milvus)
			return ctrl.Result{}, err
		}
		// Stop reconciliation as the item is being deleted
		return ctrl.Result{}, nil
	}

	// Start reconcile
	r.logger.Info("start reconcile")
	old := milvus.DeepCopy()

	if err := r.SetDefault(ctx, milvus); err != nil {
		return ctrl.Result{}, err
	}

	if !IsEqual(old.Spec, milvus.Spec) {
		diff, _ := diffObject(old, milvus)
		r.logger.Info("SetDefault: "+string(diff), "name", old.Name, "namespace", old.Namespace)
		return ctrl.Result{}, r.Update(ctx, milvus)
	}

	updated, err := r.SetDefaultStatus(ctx, milvus)
	if updated || err != nil {
		return ctrl.Result{}, err
	}

	if err := r.ReconcileAll(ctx, *milvus); err != nil {
		return ctrl.Result{}, err
	}

	// status will be updated by syncer

	if milvus.Status.Status == milvusv1alpha1.StatusUnHealthy {
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	if config.IsDebug() {
		diff, err := client.MergeFrom(old).Data(milvus)
		if err != nil {
			r.logger.Info("Update diff", "diff", string(diff))
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MilvusReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&milvusv1alpha1.Milvus{}).
		Complete(r)
}
