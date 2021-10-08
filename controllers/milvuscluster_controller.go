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

	"github.com/go-logr/logr"
	"helm.sh/helm/v3/pkg/cli"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/milvus-io/milvus-operator/api/v1alpha1"
	milvusiov1alpha1 "github.com/milvus-io/milvus-operator/api/v1alpha1"
	"github.com/milvus-io/milvus-operator/pkg/config"
)

const (
	MCFinalizerName = "milvuscluster.milvus.io/finalizer"
)

// MilvusClusterReconciler reconciles a MilvusCluster object
type MilvusClusterReconciler struct {
	client.Client
	Scheme       *runtime.Scheme
	logger       logr.Logger
	helmSettings *cli.EnvSettings
}

func NewMilvusClusterReconciler(client client.Client, scheme *runtime.Scheme, settings *cli.EnvSettings) *MilvusClusterReconciler {
	return &MilvusClusterReconciler{
		Client:       client,
		Scheme:       scheme,
		logger:       ctrl.Log.WithName("controller").WithName("milvus"),
		helmSettings: settings,
	}
}

//+kubebuilder:rbac:groups=milvus.io,resources=milvusclusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=milvus.io,resources=milvusclusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=milvus.io,resources=milvusclusters/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=statefulsets;deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=serviceaccounts,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="policy",resources=poddisruptionbudgets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="policy",resources=podsecuritypolicies,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources=rolebindings,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources=roles,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources=clusterrolebindings,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources=clusterroles,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="networking.k8s.io",resources=ingresses,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="extensions",resources=ingresses,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *MilvusClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	if !config.IsDebug() {
		defer func() {
			if err := recover(); err != nil {
				r.logger.Error(err.(error), "reconcile panic")
			}
		}()
	}

	milvuscluster := &milvusiov1alpha1.MilvusCluster{}
	if err := r.Get(ctx, req.NamespacedName, milvuscluster); err != nil {
		if errors.IsNotFound(err) {
			// The resource may have be deleted after reconcile request coming in
			// Reconcile is done
			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, fmt.Errorf("error get milvus cluster: %w", err)
	}

	// Finalize
	if milvuscluster.ObjectMeta.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(milvuscluster, MCFinalizerName) {
			controllerutil.AddFinalizer(milvuscluster, MCFinalizerName)
			if err := r.Update(ctx, milvuscluster); err != nil {
				return ctrl.Result{}, err
			}
		}

	} else {
		if controllerutil.ContainsFinalizer(milvuscluster, MCFinalizerName) {
			if err := r.Finalize(ctx, *milvuscluster); err != nil {
				return ctrl.Result{}, err
			}
			controllerutil.RemoveFinalizer(milvuscluster, MCFinalizerName)
			if err := r.Update(ctx, milvuscluster); err != nil {
				return ctrl.Result{}, err
			}
		}
		// Stop reconciliation as the item is being deleted
		return ctrl.Result{}, nil
	}

	// Start reconcile
	r.logger.Info("start reconcile")
	old := milvuscluster.DeepCopy()

	if err := r.SetDefault(ctx, milvuscluster); err != nil {
		return ctrl.Result{}, err
	}

	if !IsEqual(old.Spec, milvuscluster.Spec) {
		return ctrl.Result{}, r.Update(ctx, milvuscluster)
	}

	if err := r.ReconcileAll(ctx, *milvuscluster); err != nil {
		return ctrl.Result{}, err
	}

	if err := r.UpdateStatus(ctx, milvuscluster); err != nil {
		return ctrl.Result{}, err
	}

	if milvuscluster.Status.Status == v1alpha1.StatusUnHealthy {
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	/* 	if config.IsDebug() {
		diff, err := client.MergeFrom(old).Data(milvuscluster)
		if err != nil {
			r.logger.Info("Update diff", "diff", string(diff))
		}
	} */

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MilvusClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	builder := ctrl.NewControllerManagedBy(mgr).
		For(&milvusiov1alpha1.MilvusCluster{}).
		//Owns(&appsv1.Deployment{}).
		//Owns(&corev1.ConfigMap{}).
		//Owns(&corev1.Service{}).
		//WithEventFilter(&MilvusClusterPredicate{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 1})

	if config.IsDebug() {
		builder.WithEventFilter(DebugPredicate())
	}

	return builder.Complete(r)
}

var predicateLog = logf.Log.WithName("predicates").WithName("MilvusCluster")

type MilvusClusterPredicate struct {
	predicate.Funcs
}

func (*MilvusClusterPredicate) Create(e event.CreateEvent) bool {
	if _, ok := e.Object.(*milvusiov1alpha1.MilvusCluster); !ok {
		return false
	}

	return true
}

func (*MilvusClusterPredicate) Update(e event.UpdateEvent) bool {
	if IsEqual(e.ObjectOld, e.ObjectNew) {
		obj := fmt.Sprintf("%s/%s", e.ObjectNew.GetNamespace(), e.ObjectNew.GetName())
		predicateLog.Info("Update Equal", "obj", obj, "kind", e.ObjectNew.GetObjectKind())
		return false
	}

	return true
}
