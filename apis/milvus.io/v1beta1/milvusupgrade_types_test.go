package v1beta1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestObjectReference_Object(t *testing.T) {
	s := new(ObjectReference)
	obj := s.Object()
	assert.Equal(t, s.Namespace, obj.Namespace)
	assert.Equal(t, s.Name, obj.Name)
}

func TestMilvusUpgradeState_NeedRequeue(t *testing.T) {
	s := UpgradeStateSucceeded
	assert.False(t, s.NeedRequeue())
	s = UpgradeStatePending
	assert.True(t, s.NeedRequeue())
	s = UpgradeStateOldVersionStopping
	assert.True(t, s.NeedRequeue())
}
