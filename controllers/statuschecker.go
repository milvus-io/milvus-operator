package controllers

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/minio/madmin-go"
	"go.etcd.io/etcd/api/v3/etcdserverpb"
	"go.etcd.io/etcd/api/v3/v3rpc/rpctypes"
	clientv3 "go.etcd.io/etcd/client/v3"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/milvus-io/milvus-operator/api/v1alpha1"
)

const (
	MessageEtcdReady       = "Etcd endpoints is healthy"
	MessageEtcdNotReady    = "All etcd endpoints are unhealthy"
	MessageStorageReady    = "Storage endpoints is healthy"
	MessageStorageNotReady = "All Storage endpoints are unhealthy"
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

type statusChecker struct {
	mc *v1alpha1.MilvusCluster
	client.Client
	sync.Mutex
	ctx context.Context
}

func newStatusChecker(ctx context.Context, client client.Client, mc *v1alpha1.MilvusCluster) *statusChecker {
	return &statusChecker{
		mc:     mc,
		Client: client,
		ctx:    ctx,
	}
}

func newErrStorageCondition(reason, message string) v1alpha1.MilvusClusterCondition {
	return v1alpha1.MilvusClusterCondition{
		Type:    v1alpha1.StorageReady,
		Status:  corev1.ConditionFalse,
		Reason:  reason,
		Message: message,
	}
}

func (s *statusChecker) updateCondition(c v1alpha1.MilvusClusterCondition) {
	s.Lock()
	defer s.Unlock()
	UpdateCondition(&s.mc.Status, c)
}

func (s *statusChecker) checkStorageStatus() error {
	defer func() {
		for _, c := range s.mc.Status.Conditions {
			if c.Type == v1alpha1.StorageReady && c.Status == corev1.ConditionFalse {
				s.Lock()
				s.mc.Status.StorageStatus = nil
				s.Unlock()
				return
			}
		}
	}()

	secret := &corev1.Secret{}
	key := types.NamespacedName{Namespace: s.mc.Namespace, Name: s.mc.Spec.Storage.SecretRef}
	err := s.Get(s.ctx, key, secret)
	if err != nil && !errors.IsNotFound(err) {
		return err
	}

	if errors.IsNotFound(err) {
		s.updateCondition(newErrStorageCondition(v1alpha1.ReasonSecretNotExist, MessageSecretNotExist))
		return nil
	}

	accesskey, exist1 := secret.Data[AccessKey]
	secretkey, exist2 := secret.Data[SecretKey]
	if !exist1 || !exist2 {
		s.updateCondition(newErrStorageCondition(v1alpha1.ReasonSecretNotExist, MessageKeyNotExist))
		return nil
	}

	mdmClnt, err := madmin.New(
		s.mc.Spec.Storage.Endpoint, string(accesskey), string(secretkey), s.mc.Spec.Storage.Insecure)
	if err != nil {
		s.updateCondition(newErrStorageCondition(v1alpha1.ReasonClientErr, err.Error()))
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	st, err := mdmClnt.ServerInfo(ctx)
	if err != nil {
		s.updateCondition(newErrStorageCondition(v1alpha1.ReasonClientErr, err.Error()))
		return nil
	}
	sort.Sort(madminServerSlice(st.Servers))

	s.Lock()
	s.mc.Status.StorageStatus = []v1alpha1.MilvusStorageStatus{}
	onlines := 0
	for _, server := range st.Servers {
		storageStatus := v1alpha1.MilvusStorageStatus{
			Endpoint: server.Endpoint,
			Status:   server.State,
			Uptime:   humanize.RelTime(time.Now(), time.Now().Add(time.Duration(server.Uptime)*time.Second), "", ""),
		}
		if storageStatus.Status == "online" {
			onlines++
		}
		s.mc.Status.StorageStatus = append(s.mc.Status.StorageStatus, storageStatus)
	}
	c := v1alpha1.MilvusClusterCondition{
		Type:   v1alpha1.StorageReady,
		Status: GetConditionStatus(onlines > 0),
		Reason: v1alpha1.ReasonStorageReady,
	}
	UpdateCondition(&s.mc.Status, c)
	s.Unlock()

	return nil
}

func (s *statusChecker) checkEtcdStatus() error {
	health := GetEndpointsHealth(s.mc.Spec.Etcd.Endpoints)

	s.Lock()
	defer s.Unlock()
	etcdReady := false
	s.mc.Status.EtcdStatus = []v1alpha1.MilvusEtcdStatus{}
	for _, ep := range s.mc.Spec.Etcd.Endpoints {
		epHealth := health[ep]
		if epHealth.Health {
			etcdReady = true
		}
		s.mc.Status.EtcdStatus = append(s.mc.Status.EtcdStatus, v1alpha1.MilvusEtcdStatus{
			Endpoint: epHealth.Ep,
			Healthy:  epHealth.Health,
			Error:    epHealth.Error,
		})
	}

	c := v1alpha1.MilvusClusterCondition{
		Type:    v1alpha1.EtcdReady,
		Status:  GetConditionStatus(etcdReady),
		Reason:  v1alpha1.ReasonEtcdReady,
		Message: MessageEtcdReady,
	}
	if !etcdReady {
		c.Reason = v1alpha1.ReasonEtcdNotReady
		c.Message = MessageEtcdNotReady
	}

	UpdateCondition(&s.mc.Status, c)
	return nil
}

func (s *statusChecker) checkMilvusClusterStatus() error {
	deployments := &appsv1.DeploymentList{}
	opts := &client.ListOptions{}
	opts.LabelSelector = labels.SelectorFromSet(map[string]string{
		AppLabelInstance: s.mc.Name,
		AppLabelName:     "milvus",
	})
	if err := s.Client.List(s.ctx, deployments, opts); err != nil {
		return err
	}

	ready := 0
	notReadyComponents := []string{}
	for _, deployment := range deployments.Items {
		if metav1.IsControlledBy(&deployment, s.mc) {
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

	UpdateCondition(&s.mc.Status, cond)

	if cond.Status != corev1.ConditionTrue {
		s.mc.Status.Status = v1alpha1.StatusUnHealthy
	} else {
		s.mc.Status.Status = v1alpha1.StatusHealthy
	}

	return nil
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

//
func (r *MilvusClusterReconciler) ConditionsCheck(ctx context.Context) {
	// do an initial check, then start the periodic check
	if err := r.checkConditions(ctx); err != nil {
		r.logger.Error(err, "conditionsCheck")
	}

	ticker := time.NewTicker(1 * time.Minute)
	for {
		select {
		case <-ticker.C:
			if err := r.checkConditions(ctx); err != nil {
				r.logger.Error(err, "conditionsCheck")
			}

		case <-ctx.Done():
			return
		}
	}
}

func (r *MilvusClusterReconciler) checkConditions(ctx context.Context) error {
	r.logger.Info("checkConditions start")
	milvusclusters := &v1alpha1.MilvusClusterList{}
	if err := r.List(ctx, milvusclusters, &client.ListOptions{}); err != nil {
		return err
	}

	for _, item := range milvusclusters.Items {
		item.Status.Status = v1alpha1.MilvusStatus(time.Now().String())
		r.Status().Update(ctx, &item)
	}

	r.logger.Info("checkConditions end")
	return nil
}
