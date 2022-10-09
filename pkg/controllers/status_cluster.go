package controllers

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkv1 "k8s.io/api/networking/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/milvus-io/milvus-operator/apis/milvus.io/v1beta1"
)

//go:generate mockgen -package=controllers -source=status_cluster.go -destination=status_cluster_mock.go

const (
	MessageEtcdReady         = "Etcd endpoints is healthy"
	MessageEtcdNotReady      = "All etcd endpoints are unhealthy"
	MessageStorageReady      = "Storage endpoints is healthy"
	MessageStorageNotReady   = "All Storage endpoints are unhealthy"
	MessageMsgStreamReady    = "MsgStream is ready"
	MessageMsgStreamNotReady = "MsgStream is not ready"
	MessageSecretNotExist    = "Secret not exist"
	MessageKeyNotExist       = "accesskey or secretkey not exist in secret"
	MessageDecodeErr         = "accesskey or secretkey decode error"
	MessageMilvusHealthy     = "All Milvus components are healthy"
)

var (
	S3ReadyCondition = v1beta1.MilvusCondition{
		Type:   v1beta1.StorageReady,
		Status: GetConditionStatus(true),
		Reason: v1beta1.ReasonS3Ready,
	}
)

type EtcdEndPointHealth struct {
	Ep     string `json:"endpoint"`
	Health bool   `json:"health"`
	Error  string `json:"error,omitempty"`
}

// MilvusStatusSyncerInterface abstracts MilvusStatusSyncer
type MilvusStatusSyncerInterface interface {
	RunIfNot()
	UpdateStatusForNewGeneration(ctx context.Context, mc *v1beta1.Milvus) error
}

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

var unhealthySyncInterval = 30 * time.Second

func (r *MilvusStatusSyncer) RunIfNot() {
	r.Once.Do(func() {
		go LoopWithInterval(r.ctx, r.syncUnhealthy, unhealthySyncInterval, r.logger)
		go LoopWithInterval(r.ctx, r.syncHealthy, unhealthySyncInterval*2, r.logger)
		go LoopWithInterval(r.ctx, r.updateMetrics, unhealthySyncInterval, r.logger)
	})
}

// we use global variable for the convienence of testing
var (
	healthyCount   int
	unhealthyCount int
	deletingCount  int
	creatingCount  int
)

func (r *MilvusStatusSyncer) updateMetrics() error {
	milvusList := &v1beta1.MilvusList{}
	err := r.List(r.ctx, milvusList)
	if err != nil {
		return errors.Wrap(err, "list milvus failed")
	}
	healthyCount = 0
	unhealthyCount = 0
	deletingCount = 0
	creatingCount = 0
	for i := range milvusList.Items {
		mc := &milvusList.Items[i]
		statusCollector := milvusStatusCollector.WithLabelValues(mc.Namespace, mc.Name)
		switch mc.Status.Status {
		case v1beta1.StatusHealthy:
			healthyCount++
			statusCollector.Set(MilvusStatusCodeHealthy)
		case v1beta1.StatusUnHealthy:
			unhealthyCount++
			statusCollector.Set(MilvusStatusCodeUnHealthy)
		case v1beta1.StatusDeleting:
			deletingCount++
			statusCollector.Set(MilvusStatusCodeDeleting)
		default:
			creatingCount++
			statusCollector.Set(MilvusStatusCodeCreating)
		}
	}
	milvusTotalCountCollector.WithLabelValues(string(v1beta1.StatusHealthy)).Set(float64(healthyCount))
	milvusTotalCountCollector.WithLabelValues(string(v1beta1.StatusUnHealthy)).Set(float64(unhealthyCount))
	milvusTotalCountCollector.WithLabelValues(string(v1beta1.StatusDeleting)).Set(float64(deletingCount))
	milvusTotalCountCollector.WithLabelValues(string(v1beta1.StatusCreating)).Set(float64(creatingCount))
	return nil
}

func (r *MilvusStatusSyncer) syncUnhealthy() error {
	milvusList := &v1beta1.MilvusList{}
	err := r.List(r.ctx, milvusList)
	if err != nil {
		return errors.Wrap(err, "list milvus failed")
	}
	argsArray := []Args{}
	for i := range milvusList.Items {
		mc := &milvusList.Items[i]
		// update metric
		if mc.Status.Status == "" ||
			mc.Status.Status == v1beta1.StatusHealthy ||
			mc.Status.Status == v1beta1.StatusDeleting {
			continue
		}
		argsArray = append(argsArray, Args{mc})
	}
	err = defaultGroupRunner.RunDiffArgs(r.UpdateStatusRoutine, r.ctx, argsArray)
	return errors.Wrap(err, "UpdateStatus failed")
}

func (r *MilvusStatusSyncer) syncHealthy() error {
	milvusList := &v1beta1.MilvusList{}
	err := r.List(r.ctx, milvusList)
	if err != nil {
		return errors.Wrap(err, "list milvus failed")
	}
	argsArray := []Args{}
	for i := range milvusList.Items {
		mc := &milvusList.Items[i]
		if mc.Status.Status == v1beta1.StatusHealthy {
			argsArray = append(argsArray, Args{mc})
		}
	}
	err = defaultGroupRunner.RunDiffArgs(r.UpdateStatusRoutine, r.ctx, argsArray)
	return errors.Wrap(err, "UpdateStatus failed")
}

func (r *MilvusStatusSyncer) UpdateStatusRoutine(ctx context.Context, mc *v1beta1.Milvus) error {
	// ignore if default status not set
	if !IsSetDefaultDone(mc) {
		return nil
	}
	// ObservedGeneration not up to date meaning it is being reconciled
	if mc.Status.ObservedGeneration < mc.Generation {
		return nil
	}
	// some default values may not be set if there's an upgrade
	// so we call default again to ensure
	mc.Default()

	return r.UpdateStatusForNewGeneration(ctx, mc)
}

func (r *MilvusStatusSyncer) UpdateStatusForNewGeneration(ctx context.Context, mc *v1beta1.Milvus) error {
	funcs := []Func{
		r.GetEtcdCondition,
		r.GetMinioCondition,
		r.GetMsgStreamCondition,
	}
	ress := defaultGroupRunner.RunWithResult(funcs, ctx, *mc)
	errTexts := []string{}
	for _, res := range ress {
		if res.Err == nil {
			UpdateCondition(&mc.Status, res.Data.(v1beta1.MilvusCondition))
		} else {
			errTexts = append(errTexts, res.Err.Error())
		}
	}

	if len(errTexts) > 0 {
		return fmt.Errorf("update status error: %s", strings.Join(errTexts, ":"))
	}

	milvusCond, err := GetMilvusInstanceCondition(ctx, r.Client, *mc)
	if err != nil {
		return err
	}
	UpdateCondition(&mc.Status, milvusCond)

	if milvusCond.Status != corev1.ConditionTrue {
		mc.Status.Status = v1beta1.StatusUnHealthy
	} else {
		mc.Status.Status = v1beta1.StatusHealthy
	}

	err = r.UpdateIngressStatus(ctx, mc)
	if err != nil {
		return errors.Wrap(err, "update ingress status failed")
	}

	err = replicaUpdater.UpdateReplicas(ctx, mc, r.Client)
	if err != nil {
		return errors.Wrap(err, "update replica status failed")
	}

	mc.Status.Endpoint = r.GetMilvusEndpoint(ctx, *mc)
	return r.Status().Update(ctx, mc)
}

var replicaUpdater replicaUpdaterInterface = new(replicaUpdaterImpl)

type replicaUpdaterInterface interface {
	UpdateReplicas(ctx context.Context, obj *v1beta1.Milvus, cli client.Client) error
}

type replicaUpdaterImpl struct{}

func (r replicaUpdaterImpl) UpdateReplicas(ctx context.Context, obj *v1beta1.Milvus, cli client.Client) error {
	components := GetComponentsBySpec(obj.Spec)
	status := reflect.ValueOf(obj).Elem().FieldByName("Status").Addr().Interface().(*v1beta1.MilvusStatus)
	return updateReplicas(ctx, client.ObjectKeyFromObject(obj), status, components, cli)
}

func updateReplicas(ctx context.Context, key client.ObjectKey, status *v1beta1.MilvusStatus, components []MilvusComponent, client client.Client) error {
	for _, component := range components {
		deploy, err := getComponentDeployment(ctx, key, component, client)
		if err != nil {
			return errors.Wrap(err, "get component deployment failed")
		}
		replica := 0
		if deploy != nil {
			replica = int(deploy.Status.UpdatedReplicas - deploy.Status.UnavailableReplicas)
			if replica < 0 {
				replica = 0
			}
		}
		component.SetStatusReplicas(&status.Replicas, replica)
	}
	return nil
}

func getComponentDeployment(ctx context.Context, key client.ObjectKey, component MilvusComponent, cli client.Client) (*appsv1.Deployment, error) {
	deployment := &appsv1.Deployment{}
	err := cli.Get(ctx, types.NamespacedName{Name: component.GetDeploymentName(key.Name), Namespace: key.Namespace}, deployment)
	if err != nil {
		if kerrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return deployment, nil
}

func (r *MilvusStatusSyncer) UpdateIngressStatus(ctx context.Context, mc *v1beta1.Milvus) error {
	ingress := mc.Spec.GetServiceComponent().Ingress
	if ingress == nil {
		return nil
	}
	key := client.ObjectKeyFromObject(mc)
	key.Name = key.Name + "-milvus"
	status, err := getIngressStatus(ctx, r.Client, key)
	if err != nil {
		return errors.Wrap(err, "get ingress status failed")
	}
	mc.Status.IngressStatus = *status
	return nil
}

func getIngressStatus(ctx context.Context, client client.Client, key client.ObjectKey) (*networkv1.IngressStatus, error) {
	ingress := &networkv1.Ingress{}
	err := client.Get(ctx, key, ingress)
	if err != nil {
		if kerrors.IsNotFound(err) {
			return &networkv1.IngressStatus{}, nil
		}
		return nil, err
	}
	return ingress.Status.DeepCopy(), nil
}

func (r *MilvusStatusSyncer) GetMilvusEndpoint(ctx context.Context, mc v1beta1.Milvus) string {
	info := MilvusEndpointInfo{
		Namespace:   mc.Namespace,
		Name:        mc.Name,
		ServiceType: mc.Spec.GetServiceComponent().ServiceType,
		Port:        MilvusPort,
	}
	return GetMilvusEndpoint(ctx, r.logger, r.Client, info)
}

func (r *MilvusStatusSyncer) GetMsgStreamCondition(
	ctx context.Context, mc v1beta1.Milvus) (v1beta1.MilvusCondition, error) {
	var eps = []string{}
	var getter func() v1beta1.MilvusCondition
	// rocksmq is built in, assume ok
	if mc.Spec.Dep.MsgStreamType == v1beta1.MsgStreamTypeRocksMQ {
		return msgStreamReadyCondition, nil
	}

	if mc.Spec.Dep.MsgStreamType == v1beta1.MsgStreamTypeKafka {
		getter = wrapKafkaConditonGetter(ctx, r.logger, mc.Spec.Dep.Kafka)
		eps = mc.Spec.Dep.Kafka.BrokerList
	} else {
		getter = wrapPulsarConditonGetter(ctx, r.logger, mc.Spec.Dep.Pulsar)
		eps = []string{mc.Spec.Dep.Pulsar.Endpoint}
	}
	return GetCondition(getter, eps), nil
}

// TODO: rename as GetStorageCondition
func (r *MilvusStatusSyncer) GetMinioCondition(
	ctx context.Context, mc v1beta1.Milvus) (v1beta1.MilvusCondition, error) {
	info := StorageConditionInfo{
		Namespace:   mc.Namespace,
		Bucket:      GetMinioBucket(mc.Spec.Conf.Data),
		Storage:     mc.Spec.Dep.Storage,
		UseSSL:      GetMinioSecure(mc.Spec.Conf.Data),
		UseIAM:      GetMinioUseIAM(mc.Spec.Conf.Data),
		IAMEndpoint: GetMinioIAMEndpoint(mc.Spec.Conf.Data),
	}
	getter := wrapMinioConditionGetter(ctx, r.logger, r.Client, info)
	return GetCondition(getter, []string{mc.Spec.Dep.Storage.Endpoint}), nil
}

func (r *MilvusStatusSyncer) GetEtcdCondition(ctx context.Context, mc v1beta1.Milvus) (v1beta1.MilvusCondition, error) {
	getter := wrapEtcdConditionGetter(ctx, mc.Spec.Dep.Etcd.Endpoints)
	return GetCondition(getter, mc.Spec.Dep.Etcd.Endpoints), nil
}
