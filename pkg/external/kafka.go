package external

import (
	"time"

	"github.com/Shopify/sarama"
)

func CheckKafka(brokerList []string) error {
	config := sarama.NewConfig()
	config.Net.DialTimeout = time.Second * 2
	config.Net.ReadTimeout = time.Second * 3
	const group = "milvus-operator-group"
	cli, err := sarama.NewClient(brokerList, config)
	if cli != nil {
		cli.Close()
	}
	return err
}
