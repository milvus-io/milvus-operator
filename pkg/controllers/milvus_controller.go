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

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/milvus-io/milvus-operator/apis/milvus.io/v1beta1"
	milvusv1beta1 "github.com/milvus-io/milvus-operator/apis/milvus.io/v1beta1"
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
	helmReconciler HelmReconciler
	statusSyncer   MilvusStatusSyncerInterface
}

//+kubebuilder:rbac:groups=milvus.io,resources=milvuses,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=milvus.io,resources=milvuses/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=milvus.io,resources=milvuses/finalizers,verbs=update
//+kubebuilder:rbac:groups=extensions,resources=statefulsets;deployments;pods;secrets;services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=statefulsets;deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=pods;secrets;services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=services;configmaps;secrets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=serviceaccounts,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=deployments;statefulsets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="policy",resources=poddisruptionbudgets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="policy",resources=podsecuritypolicies,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources=rolebindings,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources=roles,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources=clusterrolebindings,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources=clusterroles,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="networking.k8s.io",resources=ingresses,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="extensions",resources=ingresses,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="monitoring.coreos.com",resources=servicemonitors;podmonitors,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *MilvusReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.statusSyncer.RunIfNot()
	globalCommonInfo.InitIfNot(r.Client)

	if !config.IsDebug() {
		defer func() {
			if err := recover(); err != nil {
				r.logger.Error(err.(error), "reconcile panic")
			}
		}()
	}

	milvus := &milvusv1beta1.Milvus{}
	if err := r.Get(ctx, req.NamespacedName, milvus); err != nil {
		if errors.IsNotFound(err) {
			// The resource may have be deleted after reconcile request coming in
			// Reconcile is done
			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, fmt.Errorf("error get milvus : %w", err)
	}

	// Finalize
	if milvus.ObjectMeta.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(milvus, MilvusFinalizerName) {
			controllerutil.AddFinalizer(milvus, MilvusFinalizerName)
			err := r.Update(ctx, milvus)
			if err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		if milvus.Status.Status != milvusv1beta1.StatusDeleting {
			milvus.Status.Status = milvusv1beta1.StatusDeleting
			if err := r.Status().Update(ctx, milvus); err != nil {
				return ctrl.Result{}, err
			}
		}

		if controllerutil.ContainsFinalizer(milvus, MilvusFinalizerName) {
			if err := r.Finalize(ctx, *milvus); err != nil {
				return ctrl.Result{}, err
			}
			// metrics
			milvusStatusCollector.DeleteLabelValues(milvus.Namespace, milvus.Name)
			controllerutil.RemoveFinalizer(milvus, MilvusFinalizerName)
			err := r.Update(ctx, milvus)
			return ctrl.Result{}, err
		}
		// Stop reconciliation as the item is being deleted
		return ctrl.Result{}, nil
	}

	old := milvus.DeepCopy()
	milvus.Default()

	if !IsEqual(old.Spec, milvus.Spec) {
		diff, _ := diffObject(old, milvus)
		r.logger.Info("SetDefault: " + string(diff))
		err := r.Update(ctx, milvus)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	err := r.ReconcileLegacyValues(ctx, old, milvus)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = r.SetDefaultStatus(ctx, milvus)
	if err != nil {
		return ctrl.Result{}, err
	}

	if err := r.ReconcileAll(ctx, *milvus); err != nil {
		return ctrl.Result{}, err
	}

	milvus.Status.ObservedGeneration = milvus.Generation
	if err := r.statusSyncer.UpdateStatusForNewGeneration(ctx, milvus); err != nil {
		return ctrl.Result{}, err
	}
	// metrics
	milvusStatusCollector.WithLabelValues(milvus.Namespace, milvus.Name).Set(MilvusStatusToCode(milvus.Status.Status))

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MilvusReconciler) SetupWithManager(mgr ctrl.Manager) error {
	builder := ctrl.NewControllerManagedBy(mgr).
		For(&milvusv1beta1.Milvus{}).
		// For(&milvusv1alpha1.MilvusCluster{}).
		//Owns(&appsv1.Deployment{}).
		//Owns(&corev1.ConfigMap{}).
		//Owns(&corev1.Service{}).
		//WithEventFilter(&MilvusPredicate{}).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: config.MaxConcurrentReconcile,
		})

	/* if config.IsDebug() {
		builder.WithEventFilter(DebugPredicate())
	} */

	return builder.Complete(r)
}

var predicateLog = logf.Log.WithName("predicates").WithName("Milvus")

type MilvusPredicate struct {
	predicate.Funcs
}

func (*MilvusPredicate) Create(e event.CreateEvent) bool {
	if _, ok := e.Object.(*milvusv1beta1.Milvus); !ok {
		return false
	}

	return true
}

func (*MilvusPredicate) Update(e event.UpdateEvent) bool {
	if IsEqual(e.ObjectOld, e.ObjectNew) {
		obj := fmt.Sprintf("%s/%s", e.ObjectNew.GetNamespace(), e.ObjectNew.GetName())
		predicateLog.Info("Update Equal", "obj", obj, "kind", e.ObjectNew.GetObjectKind())
		return false
	}

	return true
}

func (r *MilvusReconciler) ReconcileLegacyValues(ctx context.Context, old, milvus *v1beta1.Milvus) error {
	if !milvus.LegacyNeedSyncValues() {
		return nil
	}

	err := r.syncLegacyValues(ctx, milvus)
	if err != nil {
		return err
	}
	diff, _ := diffObject(old, milvus)
	r.logger.Info("SyncValues: " + string(diff))
	err = r.Update(ctx, milvus)
	return err
}

func (r *MilvusReconciler) syncLegacyValues(ctx context.Context, m *v1beta1.Milvus) error {
	// sync etcd
	if !m.Spec.Dep.Etcd.External {
		releaseValues, err := r.helmReconciler.GetValues(m.Namespace, m.Name+"-etcd")
		if err != nil {
			return err
		}
		m.Spec.Dep.Etcd.InCluster.Values.Data = releaseValues
	}

	// sync mq
	switch m.Spec.Dep.MsgStreamType {
	case v1beta1.MsgStreamTypePulsar:
		if !m.Spec.Dep.Pulsar.External {
			releaseValues, err := r.helmReconciler.GetValues(m.Namespace, m.Name+"-pulsar")
			if err != nil {
				return err
			}
			m.Spec.Dep.Pulsar.InCluster.Values.Data = releaseValues
		}
	case v1beta1.MsgStreamTypeKafka:
		if !m.Spec.Dep.Kafka.External {
			releaseValues, err := r.helmReconciler.GetValues(m.Namespace, m.Name+"-kafka")
			if err != nil {
				return err
			}
			m.Spec.Dep.Kafka.InCluster.Values.Data = releaseValues
		}
	}

	// sync minio
	if !m.Spec.Dep.Storage.External {
		releaseValues, err := r.helmReconciler.GetValues(m.Namespace, m.Name+"-minio")
		if err != nil {
			return err
		}
		m.Spec.Dep.Storage.InCluster.Values.Data = releaseValues
	}

	m.SetLegacySynced()
	return nil
}
