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

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/go-logr/logr"
	milvusiov1alpha1 "github.com/milvus-io/milvus-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
)

// MilvusClusterReconciler reconciles a MilvusCluster object
type MilvusClusterReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	logger logr.Logger
}

func NewMilvusClusterReconciler(client client.Client, scheme *runtime.Scheme) *MilvusClusterReconciler {
	return &MilvusClusterReconciler{
		Client: client,
		Scheme: scheme,
		logger: ctrl.Log.WithName("controller").WithName("milvus"),
	}
}

//+kubebuilder:rbac:groups=milvus.io,resources=milvusclusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=milvus.io,resources=milvusclusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=milvus.io,resources=milvusclusters/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *MilvusClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	defer func() {
		if err := recover(); err != nil {
			r.logger.Error(err.(error), "reconcile panic")
		}
	}()

	r.logger.Info("Starting reconcile", "milvuscluster", req.NamespacedName)
	milvuscluster := &milvusiov1alpha1.MilvusCluster{}
	if err := r.Get(ctx, req.NamespacedName, milvuscluster); err != nil {
		if errors.IsNotFound(err) {
			// The resource may have be deleted after reconcile request coming in
			// Reconcile is done
			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, fmt.Errorf("error get milvus cluster: %w", err)
	}

	// Check if it is being deleted
	if !milvuscluster.ObjectMeta.DeletionTimestamp.IsZero() {
		r.logger.Info("milvus cluster is being deleted", "name", req.NamespacedName)
		return ctrl.Result{}, nil
	}

	// Start reconcile
	if err := r.ReconcileConfigMaps(ctx, milvuscluster); err != nil {
		return ctrl.Result{}, err
	}
	if err := r.ReconcileDeployments(ctx, milvuscluster); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MilvusClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&milvusiov1alpha1.MilvusCluster{}).
		Complete(r)
}
