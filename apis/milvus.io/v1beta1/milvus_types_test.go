package v1beta1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetMilvusConditionByType(t *testing.T) {
	s := &MilvusStatus{}
	ret := GetMilvusConditionByType(s, MilvusReady)
	assert.Nil(t, ret)

	s.Conditions = []MilvusCondition{
		{
			Type: MilvusReady,
		},
	}
	ret = GetMilvusConditionByType(s, MilvusReady)
	assert.NotNil(t, ret)
}
