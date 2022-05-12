package controllers

import (
	"context"
	"errors"
	"testing"

	"github.com/apache/pulsar-client-go/pulsar"
	"github.com/golang/mock/gomock"
	"github.com/milvus-io/milvus-operator/apis/milvus.io/v1alpha1"
	"github.com/minio/madmin-go"
	"github.com/stretchr/testify/assert"
	pb "go.etcd.io/etcd/api/v3/etcdserverpb"
	"go.etcd.io/etcd/api/v3/v3rpc/rpctypes"
	clientv3 "go.etcd.io/etcd/client/v3"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

func getMockPulsarNewClient(cli pulsar.Client, err error) func(options pulsar.ClientOptions) (pulsar.Client, error) {
	return func(options pulsar.ClientOptions) (pulsar.Client, error) {
		return cli, err
	}
}

func TestGetKafkaCondition(t *testing.T) {
	checkKafka = func(p v1alpha1.MilvusKafka) error { return nil }
	ret := GetKafkaCondition(context.TODO(), logf.Log.WithName("test"), v1alpha1.MilvusKafka{})
	assert.Equal(t, corev1.ConditionTrue, ret.Status)

	checkKafka = func(p v1alpha1.MilvusKafka) error { return errors.New("failed") }
	ret = GetKafkaCondition(context.TODO(), logf.Log.WithName("test"), v1alpha1.MilvusKafka{})
	assert.Equal(t, corev1.ConditionFalse, ret.Status)
}

func TestGetPulsarCondition(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.TODO()
	logger := logf.Log.WithName("test")
	mockPulsarNewClient := NewMockPulsarClient(ctrl)
	errTest := errors.New("test")

	// new client failed, no err
	pulsarNewClient = getMockPulsarNewClient(mockPulsarNewClient, errTest)
	ret := GetPulsarCondition(ctx, logger, v1alpha1.MilvusPulsar{})
	assert.Equal(t, corev1.ConditionFalse, ret.Status)
	assert.Equal(t, v1alpha1.ReasonMsgStreamNotReady, ret.Reason)

	// new client ok, create read failed, no err
	gomock.InOrder(
		mockPulsarNewClient.EXPECT().CreateReader(gomock.Any()).Return(nil, errTest),
		mockPulsarNewClient.EXPECT().Close(),
	)
	pulsarNewClient = getMockPulsarNewClient(mockPulsarNewClient, nil)
	ret = GetPulsarCondition(ctx, logger, v1alpha1.MilvusPulsar{})
	assert.Equal(t, corev1.ConditionFalse, ret.Status)
	assert.Equal(t, v1alpha1.ReasonMsgStreamNotReady, ret.Reason)

	// new client ok, create read ok, no err
	mockReader := NewMockPulsarReader(ctrl)
	gomock.InOrder(
		mockPulsarNewClient.EXPECT().CreateReader(gomock.Any()).Return(mockReader, nil),
		mockReader.EXPECT().Close(),
		mockPulsarNewClient.EXPECT().Close(),
	)
	pulsarNewClient = getMockPulsarNewClient(mockPulsarNewClient, nil)
	ret = GetPulsarCondition(ctx, logger, v1alpha1.MilvusPulsar{})
	assert.Equal(t, corev1.ConditionTrue, ret.Status)
	assert.Equal(t, v1alpha1.ReasonMsgStreamReady, ret.Reason)
}

func getMockNewMinioClientFunc(cli MinioClient, err error) NewMinioClientFunc {
	return func(endpoint string, accessKeyID, secretAccessKey string, secure bool) (MinioClient, error) {
		return cli, err
	}
}

func TestGetMinioCondition(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.TODO()
	logger := logf.Log.WithName("test")
	mockK8sCli := NewMockK8sClient(ctrl)
	mockMinio := NewMockMinioClient(ctrl)
	errTest := errors.New("test")
	errNotFound := k8sErrors.NewNotFound(schema.GroupResource{}, "")

	t.Run(`get secret failed`, func(t *testing.T) {
		defer ctrl.Finish()
		mockK8sCli.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(errTest)
		ret := GetMinioCondition(ctx, logger, mockK8sCli, StorageConditionInfo{})
		assert.Equal(t, v1alpha1.ReasonClientErr, ret.Reason)
	})

	t.Run(`secret not found`, func(t *testing.T) {
		defer ctrl.Finish()
		mockK8sCli.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(errNotFound)
		ret := GetMinioCondition(ctx, logger, mockK8sCli, StorageConditionInfo{})
		assert.Equal(t, corev1.ConditionFalse, ret.Status)
		assert.Equal(t, v1alpha1.ReasonSecretNotExist, ret.Reason)
	})

	t.Run(`secrets keys not found`, func(t *testing.T) {
		defer ctrl.Finish()
		mockK8sCli.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
		ret := GetMinioCondition(ctx, logger, mockK8sCli, StorageConditionInfo{})
		assert.Equal(t, corev1.ConditionFalse, ret.Status)
		assert.Equal(t, v1alpha1.ReasonSecretNotExist, ret.Reason)
	})

	t.Run("new client failed", func(t *testing.T) {
		defer ctrl.Finish()
		newMinioClientFunc = getMockNewMinioClientFunc(nil, errTest)
		mockK8sCli.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).
			Do(func(ctx interface{}, key interface{}, secret *corev1.Secret) {
				secret.Data = map[string][]byte{
					AccessKey: []byte("accessKeyID"),
					SecretKey: []byte("secretAccessKey"),
				}
			})
		ret := GetMinioCondition(ctx, logger, mockK8sCli, StorageConditionInfo{})
		assert.Equal(t, corev1.ConditionFalse, ret.Status)
		assert.Equal(t, v1alpha1.ReasonClientErr, ret.Reason)

	})

	t.Run("new client ok, check failed", func(t *testing.T) {
		newMinioClientFunc = getMockNewMinioClientFunc(mockMinio, nil)
		mockK8sCli.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).
			Do(func(ctx interface{}, key interface{}, secret *corev1.Secret) {
				secret.Data = map[string][]byte{
					AccessKey: []byte("accessKeyID"),
					SecretKey: []byte("secretAccessKey"),
				}
			})
		mockMinio.EXPECT().ServerInfo(gomock.Any()).Return(madmin.InfoMessage{}, errTest)
		ret := GetMinioCondition(ctx, logger, mockK8sCli, StorageConditionInfo{})
		assert.Equal(t, corev1.ConditionFalse, ret.Status)
		assert.Equal(t, v1alpha1.ReasonClientErr, ret.Reason)

	})

	t.Run("new get info ok, check failed", func(t *testing.T) {
		mockK8sCli.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).
			Do(func(ctx interface{}, key interface{}, secret *corev1.Secret) {
				secret.Data = map[string][]byte{
					AccessKey: []byte("accessKeyID"),
					SecretKey: []byte("secretAccessKey"),
				}
			})
		mockMinio.EXPECT().ServerInfo(gomock.Any()).Return(madmin.InfoMessage{}, nil)
		ret := GetMinioCondition(ctx, logger, mockK8sCli, StorageConditionInfo{})
		assert.Equal(t, corev1.ConditionFalse, ret.Status)
		assert.Equal(t, v1alpha1.ReasonStorageNotReady, ret.Reason)
	})

	// one online check ok
	t.Run(`is "not found" err`, func(t *testing.T) {
		mockK8sCli.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).
			Do(func(ctx interface{}, key interface{}, secret *corev1.Secret) {
				secret.Data = map[string][]byte{
					AccessKey: []byte("accessKeyID"),
					SecretKey: []byte("secretAccessKey"),
				}
			})
		mockMinio.EXPECT().ServerInfo(gomock.Any()).Return(madmin.InfoMessage{
			Servers: []madmin.ServerProperties{
				{State: "ok"},
				{State: "not ok"},
			},
		}, nil)
		ret := GetMinioCondition(ctx, logger, mockK8sCli, StorageConditionInfo{})
		assert.Equal(t, corev1.ConditionTrue, ret.Status)
		assert.Equal(t, v1alpha1.ReasonStorageReady, ret.Reason)
	})
}

func getMockNewEtcdClient(cli EtcdClient, err error) NewEtcdClientFunc {
	return func(cfg clientv3.Config) (EtcdClient, error) {
		return cli, err
	}
}

func TestGetEtcdCondition(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.TODO()
	errTest := errors.New("test")

	// no endpoint
	ret := GetEtcdCondition(ctx, []string{})
	assert.Equal(t, corev1.ConditionFalse, ret.Status)
	assert.Equal(t, v1alpha1.ReasonEtcdNotReady, ret.Reason)

	// new client failed
	etcdNewClient = getMockNewEtcdClient(nil, errTest)
	ret = GetEtcdCondition(ctx, []string{"etcd:2379"})
	assert.Equal(t, corev1.ConditionFalse, ret.Status)
	assert.Equal(t, v1alpha1.ReasonEtcdNotReady, ret.Reason)

	// etcd get failed
	mockEtcdCli := NewMockEtcdClient(ctrl)
	etcdNewClient = getMockNewEtcdClient(mockEtcdCli, nil)
	gomock.InOrder(
		mockEtcdCli.EXPECT().Get(gomock.Any(), gomock.Any()).Return(nil, errTest),
		mockEtcdCli.EXPECT().Close(),
	)
	ret = GetEtcdCondition(ctx, []string{"etcd:2379"})
	assert.Equal(t, corev1.ConditionFalse, ret.Status)
	assert.Equal(t, v1alpha1.ReasonEtcdNotReady, ret.Reason)

	// etcd get, err permession denied, alarm failed
	etcdNewClient = getMockNewEtcdClient(mockEtcdCli, nil)
	gomock.InOrder(
		mockEtcdCli.EXPECT().Get(gomock.Any(), gomock.Any()).Return(nil, rpctypes.ErrPermissionDenied),
		mockEtcdCli.EXPECT().AlarmList(gomock.Any()).Return(nil, errTest),
		mockEtcdCli.EXPECT().Close(),
	)
	ret = GetEtcdCondition(ctx, []string{"etcd:2379"})
	assert.Equal(t, corev1.ConditionFalse, ret.Status)
	assert.Equal(t, v1alpha1.ReasonEtcdNotReady, ret.Reason)

	// etcd get, err permession denied, no alarm ok
	etcdNewClient = getMockNewEtcdClient(mockEtcdCli, nil)
	gomock.InOrder(
		mockEtcdCli.EXPECT().Get(gomock.Any(), gomock.Any()).Return(nil, rpctypes.ErrPermissionDenied),
		mockEtcdCli.EXPECT().AlarmList(gomock.Any()).Return(&clientv3.AlarmResponse{
			Alarms: []*pb.AlarmMember{
				{Alarm: pb.AlarmType_NOSPACE},
			},
		}, nil),
		mockEtcdCli.EXPECT().Close(),
	)
	ret = GetEtcdCondition(ctx, []string{"etcd:2379"})
	assert.Equal(t, corev1.ConditionFalse, ret.Status)
	assert.Equal(t, v1alpha1.ReasonEtcdNotReady, ret.Reason)

}

func TestGetMilvusEndpoint(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockClient := NewMockK8sClient(ctrl)
	ctx := context.TODO()
	logger := logf.Log.WithName("test")

	// nodePort empty return
	info := MilvusEndpointInfo{
		Namespace:   "ns",
		Name:        "name",
		ServiceType: corev1.ServiceTypeNodePort,
		Port:        10086,
	}
	assert.Empty(t, GetMilvusEndpoint(ctx, logger, mockClient, info))

	// clusterIP
	info.ServiceType = corev1.ServiceTypeClusterIP
	assert.Equal(t, "name-milvus.ns:10086", GetMilvusEndpoint(ctx, logger, mockClient, info))

	// loadbalancer failed
	info.ServiceType = corev1.ServiceTypeLoadBalancer
	mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("test"))
	assert.Empty(t, GetMilvusEndpoint(ctx, logger, mockClient, info))

	// loadbalancer not created, empty
	info.ServiceType = corev1.ServiceTypeLoadBalancer
	mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
	assert.Empty(t, GetMilvusEndpoint(ctx, logger, mockClient, info))

	// loadbalancer
	info.ServiceType = corev1.ServiceTypeLoadBalancer
	mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Do(
		func(ctx, k interface{}, v *corev1.Service) {
			v.Status.LoadBalancer.Ingress = []corev1.LoadBalancerIngress{
				{IP: "1.1.1.1"},
			}
		})
	assert.Equal(t, "1.1.1.1:10086", GetMilvusEndpoint(ctx, logger, mockClient, info))

}

func TestGetMilvusInstanceCondition(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockClient := NewMockK8sClient(ctrl)
	ctx := context.TODO()

	inst := metav1.ObjectMeta{
		Namespace: "ns",
		Name:      "mc",
		UID:       "uid",
	}

	// dependency not ready
	info := MilvusConditionInfo{
		Object:     &inst,
		Conditions: []v1alpha1.MilvusCondition{},
		IsCluster:  true,
	}
	ret, err := GetMilvusInstanceCondition(ctx, mockClient, info)
	assert.NoError(t, err)
	assert.Equal(t, corev1.ConditionFalse, ret.Status)
	assert.Equal(t, v1alpha1.ReasonDependencyNotReady, ret.Reason)

	// dependency ready, list failed, err
	info = MilvusConditionInfo{
		Object: &inst,
		Conditions: []v1alpha1.MilvusCondition{
			{Type: v1alpha1.EtcdReady, Status: corev1.ConditionTrue},
			{Type: v1alpha1.StorageReady, Status: corev1.ConditionTrue},
		},
		IsCluster: false,
	}
	mockClient.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("test"))
	ret, err = GetMilvusInstanceCondition(ctx, mockClient, info)
	assert.Error(t, err)

	// dependency ready, standalone one ok
	info = MilvusConditionInfo{
		Object: &inst,
		Conditions: []v1alpha1.MilvusCondition{
			{Type: v1alpha1.EtcdReady, Status: corev1.ConditionTrue},
			{Type: v1alpha1.StorageReady, Status: corev1.ConditionTrue},
		},
		IsCluster: false,
	}

	trueVal := true
	mockClient.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any()).
		Do(func(ctx interface{}, list *appsv1.DeploymentList, opts interface{}) {
			list.Items = []appsv1.Deployment{

				{},
			}
			list.Items[0].OwnerReferences = []metav1.OwnerReference{
				{Controller: &trueVal, UID: "uid"},
			}
			list.Items[0].Status.Conditions = []appsv1.DeploymentCondition{
				{Type: appsv1.DeploymentAvailable, Status: corev1.ConditionTrue},
			}
		})
	ret, err = GetMilvusInstanceCondition(ctx, mockClient, info)
	assert.NoError(t, err)
	assert.Equal(t, corev1.ConditionTrue, ret.Status)

	// dependency ready, cluster 8 ok
	info = MilvusConditionInfo{
		Object: &inst,
		Conditions: []v1alpha1.MilvusCondition{
			{Type: v1alpha1.EtcdReady, Status: corev1.ConditionTrue},
			{Type: v1alpha1.StorageReady, Status: corev1.ConditionTrue},
			{Type: v1alpha1.MsgStreamReady, Status: corev1.ConditionTrue},
		},
		IsCluster: true,
	}
	mockClient.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any()).
		Do(func(ctx interface{}, list *appsv1.DeploymentList, opts interface{}) {
			list.Items = []appsv1.Deployment{
				{}, {}, {}, {},
				{}, {}, {}, {},
			}
			for i := 0; i < 8; i++ {
				list.Items[i].OwnerReferences = []metav1.OwnerReference{
					{Controller: &trueVal, UID: "uid"},
				}
				list.Items[i].Status.Conditions = []appsv1.DeploymentCondition{
					{Type: appsv1.DeploymentAvailable, Status: corev1.ConditionTrue},
				}
			}
		})
	ret, err = GetMilvusInstanceCondition(ctx, mockClient, info)
	assert.NoError(t, err)
	assert.Equal(t, corev1.ConditionTrue, ret.Status)

	// dependency ready, cluster 1 fail, fail
	mockClient.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any()).
		Do(func(ctx interface{}, list *appsv1.DeploymentList, opts interface{}) {
			list.Items = []appsv1.Deployment{
				{}, {}, {}, {},
				{}, {}, {}, {},
			}
			for i := 0; i < 8; i++ {
				list.Items[i].OwnerReferences = []metav1.OwnerReference{
					{Controller: &trueVal, UID: "uid"},
				}
				list.Items[i].Status.Conditions = []appsv1.DeploymentCondition{
					{Type: appsv1.DeploymentAvailable, Status: corev1.ConditionTrue},
				}
			}
			list.Items[7].Status.Conditions = []appsv1.DeploymentCondition{
				{Type: appsv1.DeploymentAvailable, Status: corev1.ConditionFalse},
			}
		})
	ret, err = GetMilvusInstanceCondition(ctx, mockClient, info)
	assert.NoError(t, err)
	assert.Equal(t, corev1.ConditionFalse, ret.Status)
}
