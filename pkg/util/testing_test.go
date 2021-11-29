package util

import (
	"testing"

	"github.com/milvus-io/milvus-operator/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestGetGitRepoRootDir(t *testing.T) {
	repoRoot := GetGitRepoRootDir()
	assert.NoError(t, config.Init(repoRoot))
}
