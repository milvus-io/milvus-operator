package external

import (
	"time"

	"github.com/Shopify/sarama"
	"github.com/milvus-io/milvus-operator/apis/milvus.io/v1beta1"
)

func CheckKafka(p v1beta1.MilvusKafka) error {
	config := sarama.NewConfig()
	config.Net.DialTimeout = time.Second * 2
	config.Net.ReadTimeout = time.Second * 3
	const group = "milvus-operator-group"
	cli, err := sarama.NewClient(p.BrokerList, config)
	if cli != nil {
		cli.Close()
	}
	return err
}
