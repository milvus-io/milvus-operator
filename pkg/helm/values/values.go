package values

import (
	"io/ioutil"

	"github.com/pkg/errors"
	"sigs.k8s.io/yaml"
)

type DependencyKind string
type Chart = string
type Values = map[string]interface{}

const (
	DependencyKindEtcd    DependencyKind = "Etcd"
	DependencyKindStorage DependencyKind = "Storage"
	DependencyKindPulsar  DependencyKind = "Pulsar"
	DependencyKindKafka   DependencyKind = "Kafka"

	Etcd   = "etcd"
	Minio  = "minio"
	Pulsar = "pulsar"
	Kafka  = "kafka"
)

var (
	// DefaultValuesPath is the path to the default values file
	// variable in test, const in runtime
	DefaultValuesPath = "config/assets/charts/values.yaml"
)

type DefaultValuesProvider interface {
	GetDefaultValues(dependencyName DependencyKind) map[string]interface{}
}

var globalDefaultValues DefaultValuesProvider = &dummyValues{}

func GetDefaultValuesProvider() DefaultValuesProvider {
	return globalDefaultValues
}

// DefaultValuesProviderImpl is a DefaultValuesProvider implementation
type DefaultValuesProviderImpl struct {
	chartDefaultValues map[Chart]Values
}

func MustInitDefaultValuesProvider() {
	values, err := readValuesFromFile(DefaultValuesPath)
	if err != nil {
		err = errors.Wrapf(err, "failed to read default helm chart values from [%s]", DefaultValuesPath)
		panic(err)
	}
	globalDefaultValues = &DefaultValuesProviderImpl{
		chartDefaultValues: map[Chart]Values{
			Etcd:   values[Etcd].(Values),
			Minio:  values[Minio].(Values),
			Pulsar: values[Pulsar].(Values),
			Kafka:  values[Kafka].(Values),
		},
	}
}

func (d DefaultValuesProviderImpl) GetDefaultValues(dependencyName DependencyKind) map[string]interface{} {
	switch dependencyName {
	case DependencyKindEtcd:
		return d.chartDefaultValues[Etcd]
	case DependencyKindStorage:
		return d.chartDefaultValues[Minio]
	case DependencyKindPulsar:
		return d.chartDefaultValues[Pulsar]
	case DependencyKindKafka:
		return d.chartDefaultValues[Kafka]
	default:
		panic("unknown dependency kind")
	}
}

func readValuesFromFile(file string) (Values, error) {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read file %s", file)
	}
	ret := Values{}
	err = yaml.Unmarshal(data, &ret)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal file %s", file)
	}
	return ret, nil
}
