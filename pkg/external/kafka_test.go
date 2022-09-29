package external

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCheckKafkaFailed(t *testing.T) {
	err := CheckKafka([]string{})
	assert.Error(t, err)
	err = CheckKafka([]string{"dummy:9092"})
	assert.Error(t, err)
}
