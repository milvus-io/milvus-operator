package controllers

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/apache/pulsar-client-go/pulsar"
	"github.com/go-logr/logr"
	"github.com/milvus-io/milvus-operator/apis/milvus.io/v1alpha1"
	"github.com/minio/madmin-go"
	"go.etcd.io/etcd/api/v3/v3rpc/rpctypes"
	clientv3 "go.etcd.io/etcd/client/v3"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// pulsarNewClient wraps pulsar.NewClient for test mock convenience
var pulsarNewClient = pulsar.NewClient

func GetPulsarCondition(ctx context.Context, logger logr.Logger, p v1alpha1.MilvusPulsar) (v1alpha1.MilvusCondition, error) {

	client, err := pulsarNewClient(pulsar.ClientOptions{
		URL:               "pulsar://" + p.Endpoint,
		ConnectionTimeout: 2 * time.Second,
		OperationTimeout:  3 * time.Second,
		Logger:            newPulsarLog(logger),
	})

	if err != nil {
		return newErrPulsarCondResult(v1alpha1.ReasonPulsarNotReady, err.Error()), nil
	}
	defer client.Close()

	reader, err := client.CreateReader(pulsar.ReaderOptions{
		Topic:          "milvus-operator-topic",
		StartMessageID: pulsar.EarliestMessageID(),
	})
	if err != nil {
		return newErrPulsarCondResult(v1alpha1.ReasonPulsarNotReady, err.Error()), nil
	}
	defer reader.Close()

	return v1alpha1.MilvusCondition{
		Type:    v1alpha1.PulsarReady,
		Status:  GetConditionStatus(true),
		Reason:  v1alpha1.ReasonPulsarReady,
		Message: MessagePulsarReady,
	}, nil
}

// StorageConditionInfo is info for acquiring storage condition
type StorageConditionInfo struct {
	Namespace string
	Storage   v1alpha1.MilvusStorage
	EndPoint  string
	UseSSL    bool
}

type NewMinioClientFunc func(endpoint string, accessKeyID, secretAccessKey string, secure bool) (MinioClient, error)

// newMinioClientFunc wraps madmin.New for test mock convenience
var newMinioClientFunc NewMinioClientFunc = func(endpoint string, accessKeyID, secretAccessKey string, secure bool) (MinioClient, error) {
	return madmin.New(endpoint, accessKeyID, secretAccessKey, secure)
}

func GetMinioCondition(
	ctx context.Context, logger logr.Logger, cli client.Client, info StorageConditionInfo) (v1alpha1.MilvusCondition, error) {
	secret := &corev1.Secret{}
	key := types.NamespacedName{Namespace: info.Namespace, Name: info.Storage.SecretRef}
	err := cli.Get(ctx, key, secret)
	if err != nil && !k8sErrors.IsNotFound(err) {
		return v1alpha1.MilvusCondition{}, err
	}

	if k8sErrors.IsNotFound(err) {
		return newErrStorageCondResult(v1alpha1.ReasonSecretNotExist, MessageSecretNotExist), nil
	}

	accesskey, exist1 := secret.Data[AccessKey]
	secretkey, exist2 := secret.Data[SecretKey]
	if !exist1 || !exist2 {
		return newErrStorageCondResult(v1alpha1.ReasonSecretNotExist, MessageKeyNotExist), nil
	}

	mdmClnt, err := newMinioClientFunc(
		info.Storage.Endpoint,
		string(accesskey), string(secretkey),
		info.UseSSL,
	)

	if err != nil {
		return newErrStorageCondResult(v1alpha1.ReasonClientErr, err.Error()), nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	st, err := mdmClnt.ServerInfo(ctx)
	if err != nil {
		return newErrStorageCondResult(v1alpha1.ReasonClientErr, err.Error()), nil
	}
	ready := false
	for _, server := range st.Servers {
		if server.State == "ok" {
			ready = true
			break
		}
	}

	cond := v1alpha1.MilvusCondition{
		Type:   v1alpha1.StorageReady,
		Status: GetConditionStatus(ready),
		Reason: v1alpha1.ReasonStorageReady,
	}

	if !ready {
		cond.Reason = v1alpha1.ReasonStorageNotReady
		cond.Message = MessageStorageNotReady
	}

	return cond, nil
}

type EtcdConditionInfo struct {
	Endpoints []string
}

func GetEtcdCondition(ctx context.Context, endpoints []string) (v1alpha1.MilvusCondition, error) {
	health := GetEndpointsHealth(endpoints)
	etcdReady := false
	for _, ep := range endpoints {
		epHealth := health[ep]
		if epHealth.Health {
			etcdReady = true
		}
	}

	cond := v1alpha1.MilvusCondition{
		Type:    v1alpha1.EtcdReady,
		Status:  GetConditionStatus(etcdReady),
		Reason:  v1alpha1.ReasonEtcdReady,
		Message: MessageEtcdReady,
	}
	if !etcdReady {
		cond.Reason = v1alpha1.ReasonEtcdNotReady
		cond.Message = MessageEtcdNotReady // TODO: @shaoyue add detail err msg
	}

	return cond, nil
}

type NewEtcdClientFunc func(cfg clientv3.Config) (EtcdClient, error)

var etcdNewClient NewEtcdClientFunc = func(cfg clientv3.Config) (EtcdClient, error) {
	return clientv3.New(cfg)
}

func GetEndpointsHealth(endpoints []string) map[string]EtcdEndPointHealth {
	hch := make(chan EtcdEndPointHealth, len(endpoints))
	var wg sync.WaitGroup
	for _, ep := range endpoints {
		wg.Add(1)
		go func(ep string) {
			defer wg.Done()

			cli, err := etcdNewClient(clientv3.Config{
				Endpoints:   []string{ep},
				DialTimeout: 5 * time.Second,
			})
			if err != nil {
				hch <- EtcdEndPointHealth{Ep: ep, Health: false, Error: err.Error()}
				return
			}
			defer cli.Close()

			eh := EtcdEndPointHealth{Ep: ep, Health: false}
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			_, err = cli.Get(ctx, "health")
			// permission denied is OK since proposal goes through consensus to get it
			if err == nil || err == rpctypes.ErrPermissionDenied {
				eh.Health = true
			} else {
				eh.Error = err.Error()
			}

			if eh.Health {
				resp, err := cli.AlarmList(ctx)
				if err == nil && len(resp.Alarms) > 0 {
					eh.Health = false
					eh.Error = "Active Alarm(s): "
					for _, v := range resp.Alarms {
						eh.Error += eh.Error + v.Alarm.String()
					}
				} else if err != nil {
					eh.Health = false
					eh.Error = "Unable to fetch the alarm list"
				}

			}
			cancel()
			hch <- eh
		}(ep)
	}

	wg.Wait()
	close(hch)
	health := map[string]EtcdEndPointHealth{}
	for h := range hch {
		health[h.Ep] = h
	}

	return health
}

func newErrStorageCondResult(reason, message string) v1alpha1.MilvusCondition {
	return v1alpha1.MilvusCondition{
		Type:    v1alpha1.StorageReady,
		Status:  corev1.ConditionFalse,
		Reason:  reason,
		Message: message,
	}
}

func newErrPulsarCondResult(reason, message string) v1alpha1.MilvusCondition {
	return v1alpha1.MilvusCondition{
		Type:    v1alpha1.PulsarReady,
		Status:  corev1.ConditionFalse,
		Reason:  reason,
		Message: message,
	}
}

// MilvusEndpointInfo info for calculate the endpoint
type MilvusEndpointInfo struct {
	Namespace   string
	Name        string
	ServiceType corev1.ServiceType
	Port        int32
}

func GetMilvusEndpoint(ctx context.Context, logger logr.Logger, client client.Client, info MilvusEndpointInfo) string {
	if info.ServiceType == corev1.ServiceTypeLoadBalancer {
		proxy := &corev1.Service{}
		key := NamespacedName(info.Namespace, GetServiceInstanceName(info.Name))
		if err := client.Get(ctx, key, proxy); err != nil {
			logger.Error(err, "Get Milvus endpoint error")
			return ""
		}

		if len(proxy.Status.LoadBalancer.Ingress) < 1 {
			return ""
		}
		return fmt.Sprintf("%s:%d", proxy.Status.LoadBalancer.Ingress[0].IP, info.Port)
	}

	if info.ServiceType == corev1.ServiceTypeClusterIP {
		return fmt.Sprintf("%s-milvus.%s:%d", info.Name, info.Namespace, info.Port)
	}

	return ""
}

// MilvusConditionInfo info for calculate the milvus condition
type MilvusConditionInfo struct {
	Object     metav1.Object
	Conditions []v1alpha1.MilvusCondition
	IsCluster  bool
}

func GetMilvusInstanceCondition(ctx context.Context, cli client.Client, info MilvusConditionInfo) (v1alpha1.MilvusCondition, error) {
	if !IsDependencyReady(info.Conditions, info.IsCluster) {
		return v1alpha1.MilvusCondition{
			Type:    v1alpha1.MilvusReady,
			Status:  corev1.ConditionFalse,
			Reason:  v1alpha1.ReasonDependencyNotReady,
			Message: "Milvus Dependencies is not ready",
		}, nil
	}

	deployments := &appsv1.DeploymentList{}
	opts := &client.ListOptions{}
	opts.LabelSelector = labels.SelectorFromSet(map[string]string{
		AppLabelInstance: info.Object.GetName(),
		AppLabelName:     "milvus",
	})
	if err := cli.List(ctx, deployments, opts); err != nil {
		return v1alpha1.MilvusCondition{}, err
	}

	ready := 0
	notReadyComponents := []string{}
	for _, deployment := range deployments.Items {
		if metav1.IsControlledBy(&deployment, info.Object) {
			if DeploymentReady(deployment) {
				ready++
			} else {
				notReadyComponents = append(notReadyComponents, deployment.Labels[AppLabelComponent])
			}
		}
	}

	cond := v1alpha1.MilvusCondition{
		Type: v1alpha1.MilvusReady,
	}

	readyNeeded := 1
	if info.IsCluster {
		readyNeeded = len(MilvusComponents)
	}

	if ready >= readyNeeded {
		cond.Status = corev1.ConditionTrue
		cond.Reason = v1alpha1.ReasonMilvusClusterHealthy
		cond.Message = MessageMilvusHealthy
	} else {
		cond.Status = corev1.ConditionFalse
		cond.Reason = v1alpha1.ReasonMilvusComponentNotHealthy
		sort.Strings(notReadyComponents)
		cond.Message = fmt.Sprintf("%s not ready", notReadyComponents)
	}

	return cond, nil
}
