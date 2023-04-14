package external

import (
	"context"
	"log"
	"time"

	"github.com/pkg/errors"
	"github.com/segmentio/kafka-go"
)

func CheckKafka(brokerList []string) error {
	// make a new reader that consumes from _milvus-operator, partition 0, at offset 0
	if len(brokerList) == 0 {
		return errors.New("broker list is empty")
	}
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers: brokerList,
		Topic:   "_milvus-operator",
	})
	defer r.Close()
	var checkKafka = func() error {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()
		err := r.SetOffsetAt(ctx, time.Now())
		return errors.Wrap(err, "check consume offset from borker failed")
	}
	const backOffInterval = time.Second * 1
	const maxRetry = 3
	return doWithBackoff("checkKafka", checkKafka, maxRetry, backOffInterval)
}

func doWithBackoff(name string, fn func() error, maxRetry int, backOff time.Duration) error {
	var err error
	for i := 0; i < maxRetry; i++ {
		err = fn()
		if err == nil {
			return nil
		}
		log.Printf("%s with backoff failed, retry %d, err: %v\n", name, i, err)
		time.Sleep(backOff)
	}
	return errors.Wrap(err, name+" with backoff failed")
}
