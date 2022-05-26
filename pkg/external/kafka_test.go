package external

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCheckKafkaFailed(t *testing.T) {
	err := CheckKafka([]string{"dummy"})
	assert.Error(t, err)
}
