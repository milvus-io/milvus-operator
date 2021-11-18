package controllers

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/apache/pulsar-client-go/pulsar"
	"github.com/go-logr/logr"
	"github.com/minio/madmin-go"
	"github.com/pkg/errors"
	"go.etcd.io/etcd/api/v3/etcdserverpb"
	"go.etcd.io/etcd/api/v3/v3rpc/rpctypes"
	clientv3 "go.etcd.io/etcd/client/v3"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/milvus-io/milvus-operator/api/v1alpha1"
	milvusv1alpha1 "github.com/milvus-io/milvus-operator/api/v1alpha1"
)

const (
	MessageEtcdReady       = "Etcd endpoints is healthy"
	MessageEtcdNotReady    = "All etcd endpoints are unhealthy"
	MessageStorageReady    = "Storage endpoints is healthy"
	MessageStorageNotReady = "All Storage endpoints are unhealthy"
	MessagePulsarReady     = "Pulsar is ready"
	MessagePulsarNotReady  = "Pulsar is not ready"
	MessageSecretNotExist  = "Secret not exist"
	MessageKeyNotExist     = "accesskey or secretkey not exist in secret"
	MessageDecodeErr       = "accesskey or secretkey decode error"
	MessageMilvusHealthy   = "All Milvus components are healthy"
)

type madminServerSlice []madmin.ServerProperties

func (s madminServerSlice) Len() int           { return len(s) }
func (s madminServerSlice) Less(i, j int) bool { return s[i].Endpoint < s[j].Endpoint }
func (s madminServerSlice) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

type EtcdEndPointHealth struct {
	Ep     string `json:"endpoint"`
	Health bool   `json:"health"`
	Error  string `json:"error,omitempty"`
}

type condResult struct {
	cond v1alpha1.MilvusClusterCondition
	err  error
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

func (r *MilvusStatusSyncer) RunIfNot() {
	go r.Once.Do(func() {
		const runInterval = time.Minute * 1
		var quickInterval = runInterval / 2
		ticker := time.NewTicker(quickInterval)
		for {
			err := r.syncOneRound(r.ctx)
			if err != nil {
				r.logger.Error(err, "sync one round")
			}

			select {
			case <-r.ctx.Done():
				return
			case <-ticker.C:
			}

			err = r.syncUnhealthy(r.ctx)
			if err != nil {
				r.logger.Error(err, "sync unhealthy")
			}

			select {
			case <-r.ctx.Done():
				return
			case <-ticker.C:
			}
		}
	})
}

func WrappedUpdateStatus(
	f func(ctx context.Context, mc *v1alpha1.MilvusCluster) error,
	ctx context.Context, mc *v1alpha1.MilvusCluster) func() error {
	return func() error {
		return f(ctx, mc)
	}
}

func (r *MilvusStatusSyncer) syncUnhealthy(ctx context.Context) error {
	r.logger.Info("sync unhealthy")
	defer r.logger.Info("sync unhealthy end")
	milvusClusterList := &v1alpha1.MilvusClusterList{}
	err := r.List(ctx, milvusClusterList)
	if err != nil {
		return errors.Wrap(err, "list milvuscluster failed")
	}
	g, gtx := NewGroup(ctx)
	for i := range milvusClusterList.Items {
		if milvusClusterList.Items[i].Status.Status == milvusv1alpha1.StatusUnHealthy {
			g.Go(WrappedUpdateStatus(r.UpdateStatus, gtx, &milvusClusterList.Items[i]))
		}
	}
	if err := g.Wait(); err != nil {
		return fmt.Errorf("UpdateStatus: %w", err)
	}
	return nil
}

func (r *MilvusStatusSyncer) syncOneRound(ctx context.Context) error {
	r.logger.Info("sync one round")
	defer r.logger.Info("sync one round end")
	milvusClusterList := &v1alpha1.MilvusClusterList{}
	err := r.List(ctx, milvusClusterList)
	if err != nil {
		return errors.Wrap(err, "list milvuscluster failed")
	}
	g, gtx := NewGroup(ctx)
	for i := range milvusClusterList.Items {
		g.Go(WrappedUpdateStatus(r.UpdateStatus, gtx, &milvusClusterList.Items[i]))
	}
	if err := g.Wait(); err != nil {
		return fmt.Errorf("UpdateStatus: %w", err)
	}
	return nil
}

func (r *MilvusStatusSyncer) UpdateStatus(ctx context.Context, mc *v1alpha1.MilvusCluster) error {
	condChan := make(chan condResult, 3)
	var wait sync.WaitGroup

	// ignore if default status not set
	if !IsSetDefaultDone(mc) {
		return nil
	}

	wait.Add(1)
	go func(ch chan<- condResult, w *sync.WaitGroup) {
		defer w.Done()
		ch <- r.GetEtcdCondition(ctx, *mc)
	}(condChan, &wait)

	wait.Add(1)
	go func(ch chan<- condResult, w *sync.WaitGroup) {
		defer w.Done()
		ch <- r.GetMinioCondition(ctx, *mc)
	}(condChan, &wait)

	wait.Add(1)
	go func(ch chan<- condResult, w *sync.WaitGroup) {
		defer w.Done()
		ch <- r.GetPulsarCondition(ctx, *mc)
	}(condChan, &wait)

	wait.Wait()
	close(condChan)
	errTexts := []string{}
	for cond := range condChan {
		if cond.err == nil {
			UpdateCondition(&mc.Status, cond.cond)
		} else {
			errTexts = append(errTexts, cond.err.Error())
		}
	}

	if len(errTexts) > 0 {
		return fmt.Errorf("update status error: %s", strings.Join(errTexts, ":"))
	}

	milvusCond := r.GetMilvusClusterCondition(ctx, *mc)
	if milvusCond.err != nil {
		return milvusCond.err
	}
	UpdateCondition(&mc.Status, milvusCond.cond)

	if milvusCond.cond.Status != corev1.ConditionTrue {
		mc.Status.Status = v1alpha1.StatusUnHealthy
	} else {
		mc.Status.Status = v1alpha1.StatusHealthy
	}

	mc.Status.Endpoint = r.GetMilvusEndpoint(ctx, *mc)
	return r.Status().Update(ctx, mc)
}

func (r *MilvusStatusSyncer) GetMilvusEndpoint(ctx context.Context, mc v1alpha1.MilvusCluster) string {
	if mc.Spec.Com.Proxy.ServiceType == corev1.ServiceTypeLoadBalancer {
		proxy := &corev1.Service{}
		key := NamespacedName(mc.Namespace, Proxy.GetServiceInstanceName(mc.Name))
		if err := r.Get(ctx, key, proxy); err != nil {
			r.logger.Error(err, "Get Milvus endpoint error")
			return ""
		}

		if len(proxy.Status.LoadBalancer.Ingress) > 0 {
			return fmt.Sprintf("%s:%d", proxy.Status.LoadBalancer.Ingress[0].IP, Proxy.GetComponentPort(mc.Spec))
		}
	}

	if mc.Spec.Com.Proxy.ServiceType == corev1.ServiceTypeClusterIP {
		return fmt.Sprintf("%s-milvus.%s:%d", mc.Name, mc.Namespace, Proxy.GetComponentPort(mc.Spec))
	}

	return ""
}

func (r *MilvusStatusSyncer) GetMilvusClusterCondition(ctx context.Context, mc v1alpha1.MilvusCluster) condResult {
	if !IsDependencyReady(mc.Status) {
		return condResult{
			cond: v1alpha1.MilvusClusterCondition{
				Type:    v1alpha1.MilvusReady,
				Status:  corev1.ConditionFalse,
				Reason:  v1alpha1.ReasonDependencyNotReady,
				Message: "Milvus Dependencies is not ready",
			},
		}
	}

	deployments := &appsv1.DeploymentList{}
	opts := &client.ListOptions{}
	opts.LabelSelector = labels.SelectorFromSet(map[string]string{
		AppLabelInstance: mc.Name,
		AppLabelName:     "milvus",
	})
	if err := r.List(ctx, deployments, opts); err != nil {
		return condResult{err: err}
	}

	ready := 0
	notReadyComponents := []string{}
	for _, deployment := range deployments.Items {
		if metav1.IsControlledBy(&deployment, &mc) {
			if DeploymentReady(deployment) {
				ready++
			} else {
				notReadyComponents = append(notReadyComponents, deployment.Labels[AppLabelComponent])
			}
		}
	}

	cond := v1alpha1.MilvusClusterCondition{
		Type: v1alpha1.MilvusReady,
	}
	if ready == len(MilvusComponents) {
		cond.Status = corev1.ConditionTrue
		cond.Reason = v1alpha1.ReasonMilvusClusterHealthy
		cond.Message = MessageMilvusHealthy
	} else {
		cond.Status = corev1.ConditionFalse
		cond.Reason = v1alpha1.ReasonMilvusComponentNotHealthy
		sort.Strings(notReadyComponents)
		cond.Message = fmt.Sprintf("%s not ready", notReadyComponents)
	}

	return condResult{cond: cond}
}

func (r *MilvusStatusSyncer) GetPulsarCondition(
	ctx context.Context, mc v1alpha1.MilvusCluster) condResult {

	client, err := pulsar.NewClient(pulsar.ClientOptions{
		URL:               "pulsar://" + mc.Spec.Dep.Pulsar.Endpoint,
		ConnectionTimeout: 2 * time.Second,
		OperationTimeout:  3 * time.Second,
		Logger:            newPulsarLog(r.logger),
	})

	if err != nil {
		return newErrPulsarCondResult(v1alpha1.ReasonPulsarNotReady, err.Error())
	}
	defer client.Close()

	reader, err := client.CreateReader(pulsar.ReaderOptions{
		Topic:          "milvus-operator-topic",
		StartMessageID: pulsar.EarliestMessageID(),
	})
	if err != nil {
		return newErrPulsarCondResult(v1alpha1.ReasonPulsarNotReady, err.Error())
	}
	defer reader.Close()

	return condResult{
		cond: v1alpha1.MilvusClusterCondition{
			Type:    v1alpha1.PulsarReady,
			Status:  GetConditionStatus(true),
			Reason:  v1alpha1.ReasonPulsarReady,
			Message: MessagePulsarReady,
		},
	}
}

func (r *MilvusStatusSyncer) GetMinioCondition(
	ctx context.Context, mc v1alpha1.MilvusCluster) condResult {
	secret := &corev1.Secret{}
	key := types.NamespacedName{Namespace: mc.Namespace, Name: mc.Spec.Dep.Storage.SecretRef}
	err := r.Get(ctx, key, secret)
	if err != nil && !k8sErrors.IsNotFound(err) {
		return condResult{err: err}
	}

	if k8sErrors.IsNotFound(err) {
		return newErrStorageCondResult(v1alpha1.ReasonSecretNotExist, MessageSecretNotExist)
	}

	accesskey, exist1 := secret.Data[AccessKey]
	secretkey, exist2 := secret.Data[SecretKey]
	if !exist1 || !exist2 {
		return newErrStorageCondResult(v1alpha1.ReasonSecretNotExist, MessageKeyNotExist)
	}

	mdmClnt, err := madmin.New(
		mc.Spec.Dep.Storage.Endpoint,
		string(accesskey), string(secretkey),
		GetMinioSecure(mc.Spec.Conf.Data),
	)

	if err != nil {
		return newErrStorageCondResult(v1alpha1.ReasonClientErr, err.Error())
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	st, err := mdmClnt.ServerInfo(ctx)
	if err != nil {
		return newErrStorageCondResult(v1alpha1.ReasonClientErr, err.Error())
	}

	ready := false
	for _, server := range st.Servers {
		if server.State == "online" {
			ready = true
			break
		}
	}

	cond := v1alpha1.MilvusClusterCondition{
		Type:   v1alpha1.StorageReady,
		Status: GetConditionStatus(ready),
		Reason: v1alpha1.ReasonStorageReady,
	}

	if !ready {
		cond.Reason = v1alpha1.ReasonStorageNotReady
		cond.Message = MessageStorageNotReady
	}

	return condResult{cond: cond}
}

func (r *MilvusStatusSyncer) GetEtcdCondition(ctx context.Context, mc v1alpha1.MilvusCluster) condResult {
	endpoints := mc.Spec.Dep.Etcd.Endpoints
	health := GetEndpointsHealth(endpoints)
	etcdReady := false
	for _, ep := range endpoints {
		epHealth := health[ep]
		if epHealth.Health {
			etcdReady = true
		}
	}

	cond := v1alpha1.MilvusClusterCondition{
		Type:    v1alpha1.EtcdReady,
		Status:  GetConditionStatus(etcdReady),
		Reason:  v1alpha1.ReasonEtcdReady,
		Message: MessageEtcdReady,
	}
	if !etcdReady {
		cond.Reason = v1alpha1.ReasonEtcdNotReady
		cond.Message = MessageEtcdNotReady
	}

	return condResult{cond: cond}
}

func GetEndpointsHealth(endpoints []string) map[string]EtcdEndPointHealth {
	hch := make(chan EtcdEndPointHealth, len(endpoints))
	var wg sync.WaitGroup
	for _, ep := range endpoints {
		wg.Add(1)
		go func(ep string) {
			defer wg.Done()

			cli, err := clientv3.New(clientv3.Config{
				Endpoints:   []string{ep},
				DialTimeout: 5 * time.Second,
			})
			defer cli.Close()

			if err != nil {
				hch <- EtcdEndPointHealth{Ep: ep, Health: false, Error: err.Error()}
				return
			}

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
						switch v.Alarm {
						case etcdserverpb.AlarmType_NOSPACE:
							eh.Error = eh.Error + "NOSPACE "
						case etcdserverpb.AlarmType_CORRUPT:
							eh.Error = eh.Error + "CORRUPT "
						default:
							eh.Error = eh.Error + "UNKNOWN "
						}
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

func newErrStorageCondResult(reason, message string) condResult {
	return condResult{
		cond: v1alpha1.MilvusClusterCondition{
			Type:    v1alpha1.StorageReady,
			Status:  corev1.ConditionFalse,
			Reason:  reason,
			Message: message,
		},
	}
}

func newErrPulsarCondResult(reason, message string) condResult {
	return condResult{
		cond: v1alpha1.MilvusClusterCondition{
			Type:    v1alpha1.PulsarReady,
			Status:  corev1.ConditionFalse,
			Reason:  reason,
			Message: message,
		},
	}
}
