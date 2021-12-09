package controllers

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/milvus-io/milvus-operator/apis/milvus.io/v1alpha1"
	milvusv1alpha1 "github.com/milvus-io/milvus-operator/apis/milvus.io/v1alpha1"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type MilvusStatusSyncer struct {
	ctx context.Context
	client.Client
	logger logr.Logger

	sync.Once
}

func NewMilvusStatusSyncer(ctx context.Context, client client.Client, logger logr.Logger) *MilvusStatusSyncer {
	return &MilvusStatusSyncer{
		ctx:    ctx,
		Client: client,
		logger: logger,
	}
}

func (r *MilvusStatusSyncer) RunIfNot() {
	r.Once.Do(func() {
		go LoopWithInterval(r.ctx, r.syncUnhealthy, 30*time.Second, r.logger)
		go LoopWithInterval(r.ctx, r.syncHealthy, 1*time.Minute, r.logger)
	})
}

func WrappedUpdateStatus(
	f func(ctx context.Context, mil *v1alpha1.Milvus) error,
	ctx context.Context, mil *v1alpha1.Milvus) func() error {
	return func() error {
		return f(ctx, mil)
	}
}

func (r *MilvusStatusSyncer) syncUnhealthy() error {
	milvusList := &v1alpha1.MilvusList{}
	err := r.List(r.ctx, milvusList)
	if err != nil {
		return errors.Wrap(err, "list milvuscluster failed")
	}
	argsArray := []Args{}
	for i := range milvusList.Items {
		mil := &milvusList.Items[i]
		if mil.Status.Status == "" ||
			mil.Status.Status == milvusv1alpha1.StatusHealthy {
			continue
		}
		argsArray = append(argsArray, Args{mil})
	}
	err = defaultGroupRunner.RunDiffArgs(r.UpdateStatus, r.ctx, argsArray)
	return errors.Wrap(err, "UpdateStatus")
}

func (r *MilvusStatusSyncer) syncHealthy() error {
	milvusList := &v1alpha1.MilvusList{}
	err := r.List(r.ctx, milvusList)
	if err != nil {
		return errors.Wrap(err, "list milvuscluster failed")
	}
	argsArray := []Args{}
	for i := range milvusList.Items {
		mil := &milvusList.Items[i]
		if mil.Status.Status == milvusv1alpha1.StatusHealthy {
			argsArray = append(argsArray, Args{mil})
		}
	}
	err = defaultGroupRunner.RunDiffArgs(r.UpdateStatus, r.ctx, argsArray)
	return errors.Wrap(err, "UpdateStatus")
}

func (r *MilvusStatusSyncer) UpdateStatus(ctx context.Context, mil *v1alpha1.Milvus) error {
	// ignore if default status not set
	if !IsSetDefaultDone(mil) {
		return nil
	}

	funcs := []Func{
		r.GetEtcdCondition,
		r.GetMinioCondition,
	}
	ress := defaultGroupRunner.RunWithResult(funcs, ctx, *mil)

	errTexts := []string{}
	for _, res := range ress {
		if res.Err == nil {
			UpdateCondition(&mil.Status, res.Data.(v1alpha1.MilvusCondition))
		} else {
			errTexts = append(errTexts, res.Err.Error())
		}
	}

	if len(errTexts) > 0 {
		return fmt.Errorf("update status error: %s", strings.Join(errTexts, ":"))
	}

	milvusCond, err := r.GetMilvusCondition(ctx, *mil)
	if err != nil {
		return err
	}
	UpdateCondition(&mil.Status, milvusCond)

	if milvusCond.Status != corev1.ConditionTrue {
		mil.Status.Status = v1alpha1.StatusUnHealthy
	} else {
		mil.Status.Status = v1alpha1.StatusHealthy
	}

	mil.Status.Endpoint = r.GetMilvusEndpoint(ctx, *mil)
	return r.Status().Update(ctx, mil)
}

func (r *MilvusStatusSyncer) GetMilvusEndpoint(ctx context.Context, mil v1alpha1.Milvus) string {
	info := MilvusEndpointInfo{
		Namespace:   mil.Namespace,
		Name:        mil.Name,
		ServiceType: mil.Spec.ServiceType,
		Port:        MilvusPort, // TODO: @shaoyue: port should be configurable
	}
	return GetMilvusEndpoint(ctx, r.logger, r.Client, info)
}

func (r *MilvusStatusSyncer) GetMilvusCondition(ctx context.Context, mil v1alpha1.Milvus) (v1alpha1.MilvusCondition, error) {
	info := MilvusConditionInfo{
		Object:     mil.GetObjectMeta(),
		Conditions: mil.Status.Conditions,
		IsCluster:  false,
	}
	return GetMilvusInstanceCondition(ctx, r.Client, info)
}

func (r *MilvusStatusSyncer) GetMinioCondition(
	ctx context.Context, mil v1alpha1.Milvus) (v1alpha1.MilvusCondition, error) {
	info := StorageConditionInfo{
		Namespace: mil.Namespace,
		Storage:   mil.Spec.Dep.Storage,
		EndPoint:  mil.Spec.Dep.Storage.Endpoint,
		UseSSL:    GetMinioSecure(mil.Spec.Conf.Data),
	}
	return GetMinioCondition(ctx, r.logger, r.Client, info)
}

func (r *MilvusStatusSyncer) GetEtcdCondition(ctx context.Context, mil v1alpha1.Milvus) (v1alpha1.MilvusCondition, error) {
	return GetEtcdCondition(ctx, mil.Spec.Dep.Etcd.Endpoints)
}
