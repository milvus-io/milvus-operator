package controllers

import (
	"context"
	"errors"
	"testing"

	"github.com/apache/pulsar-client-go/pulsar"
	"github.com/golang/mock/gomock"
	"github.com/milvus-io/milvus-operator/apis/milvus.io/v1beta1"
	"github.com/milvus-io/milvus-operator/pkg/external"
	"github.com/stretchr/testify/assert"
	pb "go.etcd.io/etcd/api/v3/etcdserverpb"
	"go.etcd.io/etcd/api/v3/v3rpc/rpctypes"
	clientv3 "go.etcd.io/etcd/client/v3"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var readyDeployStatus = appsv1.DeploymentStatus{
	Conditions: []appsv1.DeploymentCondition{
		{
			Type:   appsv1.DeploymentProgressing,
			Status: corev1.ConditionTrue,
			Reason: v1beta1.NewReplicaSetAvailableReason,
		},
		{
			Type:   appsv1.DeploymentAvailable,
			Status: corev1.ConditionTrue,
		},
	},
	ObservedGeneration: 1,
	Replicas:           1,
}

func TestGetCondition(t *testing.T) {
	bak := endpointCheckCache
	defer func() { endpointCheckCache = bak }()

	t.Run("use cache", func(t *testing.T) {
		condition := v1beta1.MilvusCondition{Reason: "test"}
		endpointCheckCache = &mockEndpointCheckCache{condition: &condition, isUpToDate: true}
		ret := GetCondition(mockConditionGetter, []string{})
		assert.Equal(t, condition, ret)
	})
	t.Run("not use cache", func(t *testing.T) {
		endpointCheckCache = &mockEndpointCheckCache{condition: nil, isUpToDate: false}
		ret := GetCondition(mockConditionGetter, []string{})
		assert.Equal(t, v1beta1.MilvusCondition{Reason: "update"}, ret)
	})
}

func TestWrapGetters(t *testing.T) {
	ctx := context.TODO()
	logger := logf.Log
	t.Run("kafka", func(t *testing.T) {
		fn := wrapKafkaConditonGetter(ctx, logger, v1beta1.MilvusKafka{})
		fn()
	})
	t.Run("pulsar", func(t *testing.T) {
		fn := wrapPulsarConditonGetter(ctx, logger, v1beta1.MilvusPulsar{})
		fn()
	})
	t.Run("etcd", func(t *testing.T) {
		fn := wrapEtcdConditionGetter(ctx, []string{})
		fn()
	})
	t.Run("minio", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		cli := NewMockK8sClient(ctrl)
		cli.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
		fn := wrapMinioConditionGetter(ctx, logger, cli, StorageConditionInfo{})
		fn()
	})
}

func getMockPulsarNewClient(cli pulsar.Client, err error) func(options pulsar.ClientOptions) (pulsar.Client, error) {
	return func(options pulsar.ClientOptions) (pulsar.Client, error) {
		return cli, err
	}
}

func TestGetKafkaCondition(t *testing.T) {
	checkKafka = func([]string) error { return nil }
	ret := GetKafkaCondition(context.TODO(), logf.Log.WithName("test"), v1beta1.MilvusKafka{})
	assert.Equal(t, corev1.ConditionTrue, ret.Status)

	checkKafka = func([]string) error { return errors.New("failed") }
	ret = GetKafkaCondition(context.TODO(), logf.Log.WithName("test"), v1beta1.MilvusKafka{})
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
	ret := GetPulsarCondition(ctx, logger, v1beta1.MilvusPulsar{})
	assert.Equal(t, corev1.ConditionFalse, ret.Status)
	assert.Equal(t, v1beta1.ReasonMsgStreamNotReady, ret.Reason)

	// new client ok, create read failed, no err
	gomock.InOrder(
		mockPulsarNewClient.EXPECT().CreateReader(gomock.Any()).Return(nil, errTest),
		mockPulsarNewClient.EXPECT().Close(),
	)
	pulsarNewClient = getMockPulsarNewClient(mockPulsarNewClient, nil)
	ret = GetPulsarCondition(ctx, logger, v1beta1.MilvusPulsar{})
	assert.Equal(t, corev1.ConditionFalse, ret.Status)
	assert.Equal(t, v1beta1.ReasonMsgStreamNotReady, ret.Reason)

	// new client ok, create read ok, no err
	mockReader := NewMockPulsarReader(ctrl)
	gomock.InOrder(
		mockPulsarNewClient.EXPECT().CreateReader(gomock.Any()).Return(mockReader, nil),
		mockReader.EXPECT().Close(),
		mockPulsarNewClient.EXPECT().Close(),
	)
	pulsarNewClient = getMockPulsarNewClient(mockPulsarNewClient, nil)
	ret = GetPulsarCondition(ctx, logger, v1beta1.MilvusPulsar{})
	assert.Equal(t, corev1.ConditionTrue, ret.Status)
	assert.Equal(t, v1beta1.ReasonMsgStreamReady, ret.Reason)
}

func getMockCheckMinIOFunc(err error) checkMinIOFunc {
	return func(external.CheckMinIOArgs) error {
		return err
	}
}

func TestGetMinioCondition(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	checkMinIObak := checkMinIO
	defer func() {
		checkMinIO = checkMinIObak
	}()

	ctx := context.TODO()
	logger := logf.Log.WithName("test")
	mockK8sCli := NewMockK8sClient(ctrl)
	errTest := errors.New("test")
	errNotFound := k8sErrors.NewNotFound(schema.GroupResource{}, "")

	t.Run(`iam not get secret`, func(t *testing.T) {
		defer ctrl.Finish()
		ret := GetMinioCondition(ctx, logger, mockK8sCli, StorageConditionInfo{UseIAM: true})
		assert.Equal(t, v1beta1.ReasonClientErr, ret.Reason)
	})

	t.Run(`get secret failed`, func(t *testing.T) {
		defer ctrl.Finish()
		mockK8sCli.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(errTest)
		ret := GetMinioCondition(ctx, logger, mockK8sCli, StorageConditionInfo{})
		assert.Equal(t, v1beta1.ReasonClientErr, ret.Reason)
		assert.Equal(t, errTest.Error(), ret.Message)
	})

	t.Run(`secret not found`, func(t *testing.T) {
		defer ctrl.Finish()
		mockK8sCli.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(errNotFound)
		ret := GetMinioCondition(ctx, logger, mockK8sCli, StorageConditionInfo{})
		assert.Equal(t, corev1.ConditionFalse, ret.Status)
		assert.Equal(t, v1beta1.ReasonSecretNotExist, ret.Reason)
	})

	t.Run(`secrets keys not found`, func(t *testing.T) {
		defer ctrl.Finish()
		mockK8sCli.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
		ret := GetMinioCondition(ctx, logger, mockK8sCli, StorageConditionInfo{})
		assert.Equal(t, corev1.ConditionFalse, ret.Status)
		assert.Equal(t, v1beta1.ReasonSecretNotExist, ret.Reason)
	})

	t.Run("new client failed", func(t *testing.T) {
		defer ctrl.Finish()
		checkMinIO = getMockCheckMinIOFunc(errTest)
		mockK8sCli.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).
			Do(func(ctx interface{}, key interface{}, secret *corev1.Secret) {
				secret.Data = map[string][]byte{
					AccessKey: []byte("accessKeyID"),
					SecretKey: []byte("secretAccessKey"),
				}
			})
		ret := GetMinioCondition(ctx, logger, mockK8sCli, StorageConditionInfo{})
		assert.Equal(t, corev1.ConditionFalse, ret.Status)
		assert.Equal(t, v1beta1.ReasonClientErr, ret.Reason)

	})

	t.Run("new client ok, check ok", func(t *testing.T) {
		checkMinIO = getMockCheckMinIOFunc(nil)
		mockK8sCli.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).
			Do(func(ctx interface{}, key interface{}, secret *corev1.Secret) {
				secret.Data = map[string][]byte{
					AccessKey: []byte("accessKeyID"),
					SecretKey: []byte("secretAccessKey"),
				}
			})
		ret := GetMinioCondition(ctx, logger, mockK8sCli, StorageConditionInfo{})
		assert.Equal(t, corev1.ConditionTrue, ret.Status)
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
		ret := GetMinioCondition(ctx, logger, mockK8sCli, StorageConditionInfo{})
		assert.Equal(t, corev1.ConditionTrue, ret.Status)
		assert.Equal(t, v1beta1.ReasonStorageReady, ret.Reason)
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
	assert.Equal(t, v1beta1.ReasonEtcdNotReady, ret.Reason)

	// new client failed
	etcdNewClient = getMockNewEtcdClient(nil, errTest)
	ret = GetEtcdCondition(ctx, []string{"etcd:2379"})
	assert.Equal(t, corev1.ConditionFalse, ret.Status)
	assert.Equal(t, v1beta1.ReasonEtcdNotReady, ret.Reason)

	// etcd get failed
	mockEtcdCli := NewMockEtcdClient(ctrl)
	etcdNewClient = getMockNewEtcdClient(mockEtcdCli, nil)
	gomock.InOrder(
		mockEtcdCli.EXPECT().Get(gomock.Any(), etcdHealthKey, gomock.Any()).Return(nil, errTest),
		mockEtcdCli.EXPECT().Close(),
	)
	ret = GetEtcdCondition(ctx, []string{"etcd:2379"})
	assert.Equal(t, corev1.ConditionFalse, ret.Status)
	assert.Equal(t, v1beta1.ReasonEtcdNotReady, ret.Reason)

	// etcd get, err permession denied, alarm failed
	etcdNewClient = getMockNewEtcdClient(mockEtcdCli, nil)
	gomock.InOrder(
		mockEtcdCli.EXPECT().Get(gomock.Any(), etcdHealthKey, gomock.Any()).Return(nil, rpctypes.ErrPermissionDenied),
		mockEtcdCli.EXPECT().AlarmList(gomock.Any()).Return(nil, errTest),
		mockEtcdCli.EXPECT().Close(),
	)
	ret = GetEtcdCondition(ctx, []string{"etcd:2379"})
	assert.Equal(t, corev1.ConditionFalse, ret.Status)
	assert.Equal(t, v1beta1.ReasonEtcdNotReady, ret.Reason)

	// etcd get, err permession denied, no alarm ok
	etcdNewClient = getMockNewEtcdClient(mockEtcdCli, nil)
	gomock.InOrder(
		mockEtcdCli.EXPECT().Get(gomock.Any(), etcdHealthKey, gomock.Any()).Return(nil, rpctypes.ErrPermissionDenied),
		mockEtcdCli.EXPECT().AlarmList(gomock.Any()).Return(&clientv3.AlarmResponse{
			Alarms: []*pb.AlarmMember{
				{Alarm: pb.AlarmType_NOSPACE},
			},
		}, nil),
		mockEtcdCli.EXPECT().Close(),
	)
	ret = GetEtcdCondition(ctx, []string{"etcd:2379"})
	assert.Equal(t, corev1.ConditionFalse, ret.Status)
	assert.Equal(t, v1beta1.ReasonEtcdNotReady, ret.Reason)

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
	milvus := &v1beta1.Milvus{
		ObjectMeta: inst,
	}
	trueVal := true

	milvus.Spec.Mode = v1beta1.MilvusModeStandalone
	milvus.Default()
	t.Run(("dependency not ready"), func(t *testing.T) {
		ret, err := GetMilvusInstanceCondition(ctx, mockClient, *milvus)
		assert.NoError(t, err)
		assert.Equal(t, corev1.ConditionFalse, ret.Status)
		assert.Equal(t, v1beta1.ReasonDependencyNotReady, ret.Reason)

		milvus.Status.Conditions = []v1beta1.MilvusCondition{
			{Type: v1beta1.EtcdReady, Status: corev1.ConditionTrue},
			{Type: v1beta1.MsgStreamReady, Status: corev1.ConditionFalse},
			{Type: v1beta1.StorageReady, Status: corev1.ConditionTrue},
		}
		ret, err = GetMilvusInstanceCondition(ctx, mockClient, *milvus)
		assert.NoError(t, err)
		assert.Equal(t, corev1.ConditionFalse, ret.Status)
	})

	t.Run(("get milvus condition error"), func(t *testing.T) {
		ret, err := GetMilvusInstanceCondition(ctx, mockClient, *milvus)
		assert.NoError(t, err)
		assert.Equal(t, corev1.ConditionFalse, ret.Status)
		assert.Equal(t, v1beta1.ReasonDependencyNotReady, ret.Reason)

		milvus.Status.Conditions = []v1beta1.MilvusCondition{
			{Type: v1beta1.EtcdReady, Status: corev1.ConditionTrue},
			{Type: v1beta1.MsgStreamReady, Status: corev1.ConditionTrue},
			{Type: v1beta1.StorageReady, Status: corev1.ConditionTrue},
		}
		mockClient.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("test"))
		ret, err = GetMilvusInstanceCondition(ctx, mockClient, *milvus)
		assert.Error(t, err)
	})

	t.Run(("standalone milvus ok"), func(t *testing.T) {
		mockClient.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any()).
			Do(func(ctx interface{}, list *appsv1.DeploymentList, opts interface{}) {
				list.Items = []appsv1.Deployment{
					{},
				}
				list.Items[0].Labels = map[string]string{
					AppLabelComponent: StandaloneName,
				}
				list.Items[0].OwnerReferences = []metav1.OwnerReference{
					{Controller: &trueVal, UID: "uid"},
				}
				list.Items[0].Status = readyDeployStatus
			})
		ret, err := GetMilvusInstanceCondition(ctx, mockClient, *milvus)
		assert.NoError(t, err)
		assert.Equal(t, corev1.ConditionTrue, ret.Status)
	})

	milvus.Spec.Mode = v1beta1.MilvusModeCluster
	milvus.Default()
	t.Run(("cluster all ok"), func(t *testing.T) {
		mockClient.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any()).
			Do(func(ctx interface{}, list *appsv1.DeploymentList, opts interface{}) {
				list.Items = []appsv1.Deployment{
					{}, {}, {}, {},
					{}, {}, {}, {},
				}
				for i := 0; i < 8; i++ {
					list.Items[i].Labels = map[string]string{
						AppLabelComponent: MilvusComponents[i].Name,
					}
					list.Items[i].OwnerReferences = []metav1.OwnerReference{
						{Controller: &trueVal, UID: "uid"},
					}
					list.Items[i].Status = readyDeployStatus
				}
			})
		ret, err := GetMilvusInstanceCondition(ctx, mockClient, *milvus)
		assert.NoError(t, err)
		assert.Equal(t, corev1.ConditionTrue, ret.Status)
	})

	milvus.Spec.Com.MixCoord = &v1beta1.MilvusMixCoord{}
	milvus.Default()
	t.Run(("cluster mixture 5 ok"), func(t *testing.T) {
		mockClient.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any()).
			Do(func(ctx interface{}, list *appsv1.DeploymentList, opts interface{}) {
				list.Items = []appsv1.Deployment{
					{}, {}, {}, {},
					{},
				}
				for i := 0; i < 5; i++ {
					list.Items[i].Labels = map[string]string{
						AppLabelComponent: MixtureComponents[i].Name,
					}
					list.Items[i].OwnerReferences = []metav1.OwnerReference{
						{Controller: &trueVal, UID: "uid"},
					}
					list.Items[i].Status = readyDeployStatus
				}
			})
		ret, err := GetMilvusInstanceCondition(ctx, mockClient, *milvus)
		assert.NoError(t, err)
		assert.Equal(t, corev1.ConditionTrue, ret.Status)
	})

	milvus.Spec.Com.MixCoord = nil
	milvus.Default()
	t.Run(("cluster 1 unready"), func(t *testing.T) {
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
						{Type: appsv1.DeploymentAvailable, Reason: v1beta1.NewReplicaSetAvailableReason, Status: corev1.ConditionTrue},
					}
					list.Items[i].Status.Replicas = 1
				}
				list.Items[7].Status.Conditions = []appsv1.DeploymentCondition{
					{Type: appsv1.DeploymentAvailable, Status: corev1.ConditionFalse},
				}
			})
		ret, err := GetMilvusInstanceCondition(ctx, mockClient, *milvus)
		assert.NoError(t, err)
		assert.Equal(t, corev1.ConditionFalse, ret.Status)
	})

}

func TestCheckMinIOFailed(t *testing.T) {
	err := checkMinIO(external.CheckMinIOArgs{})
	assert.Error(t, err)
}

func TestMakeComponentDeploymentMap(t *testing.T) {
	mc := v1beta1.Milvus{}
	deploy := appsv1.Deployment{}
	deploy.Labels = map[string]string{
		AppLabelComponent: ProxyName,
	}
	scheme, _ := v1beta1.SchemeBuilder.Build()
	ctrl.SetControllerReference(&mc, &deploy, scheme)
	ret := makeComponentDeploymentMap(mc, []appsv1.Deployment{deploy})
	assert.NotNil(t, ret[ProxyName])
}

func TestIsMilvusConditionTrueByType(t *testing.T) {
	conds := []v1beta1.MilvusCondition{}
	ret := IsMilvusConditionTrueByType(conds, v1beta1.StorageReady)
	assert.False(t, ret)

	cond := v1beta1.MilvusCondition{
		Type:   v1beta1.StorageReady,
		Status: corev1.ConditionTrue,
	}
	conds = []v1beta1.MilvusCondition{cond}
	ret = IsMilvusConditionTrueByType(conds, v1beta1.StorageReady)
	assert.True(t, ret)
}
