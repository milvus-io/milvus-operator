package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResource(t *testing.T) {
	ret := Resource("test")
	assert.Equal(t, "milvus.io", ret.Group)
	assert.Equal(t, "test", ret.Resource)
}
