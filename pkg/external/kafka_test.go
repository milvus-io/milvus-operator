package external

import (
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestCheckKafkaFailed(t *testing.T) {
	err := CheckKafka([]string{})
	assert.Error(t, err)
	err = CheckKafka([]string{"dummy:9092"})
	assert.Error(t, err)
}

func Test_doWithBackoff(t *testing.T) {
	var err error
	err = doWithBackoff("test", func() error {
		return nil
	}, 3, time.Second)
	assert.NoError(t, err)

	err = doWithBackoff("test", func() error {
		return errors.New("test error")
	}, 3, time.Second)
	assert.Error(t, err)

	t.Run("failed then success", func(t *testing.T) {
		var i int
		err = doWithBackoff("test", func() error {
			i++
			if i == 3 {
				return nil
			}
			return errors.New("test error")
		}, 3, time.Second)
		assert.NoError(t, err)
		assert.Equal(t, 3, i)
	})
}
