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
