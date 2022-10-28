package v1beta1

import (
	"testing"
	"time"

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

func Test_SetStoppedAtAnnotation_RemoveStoppedAtAnnotation(t *testing.T) {
	myTimeStr := "2020-01-01T00:00:00Z"
	myTime, _ := time.Parse(time.RFC3339, myTimeStr)
	m := &Milvus{}
	InitLabelAnnotation(m)
	m.SetStoppedAtAnnotation(myTime)
	assert.Equal(t, m.GetAnnotations()[StoppedAtAnnotation], myTimeStr)
	m.RemoveStoppedAtAnnotation()
	_, exists := m.GetAnnotations()[StoppedAtAnnotation]
	assert.False(t, exists)
}
