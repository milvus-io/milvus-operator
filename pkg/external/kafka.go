package external

import (
	"net"

	"github.com/pkg/errors"
)

// CheckKafka checks if the kafka is available
// TODO: use same client as milvus
func CheckKafka(brokerList []string) error {
	if len(brokerList) < 1 {
		return errors.Errorf("no broker")
	}
	var err error
	for _, broker := range brokerList {
		var conn net.Conn
		conn, err = net.Dial("tcp", broker)
		if conn != nil {
			defer conn.Close()
		}
		if err == nil {
			return nil
		}
	}
	return errors.Wrap(err, "no broker available, one of the err")
}
