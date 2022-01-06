package config

import (
	"os"
	"testing"

	"github.com/milvus-io/milvus-operator/pkg/util"
	"github.com/stretchr/testify/assert"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func TestGetVersion(t *testing.T) {
	assert.Equal(t, version, GetVersion())
	version = "version"
	assert.Equal(t, "version", GetVersion())
}

func TestGetCommit(t *testing.T) {
	commit = "commit"
	assert.Equal(t, commit, GetCommit())
}

func TestPrintVersionMessageOK(t *testing.T) {
	logger := ctrl.Log.WithName("test")
	ctrl.SetLogger(zap.New())
	PrintVersionMessage(logger)
}

func TestInit_NewConfigFailed(t *testing.T) {
	workDir := "/a-bad-path/a-bad-path"
	err := Init(workDir)
	assert.Error(t, err)
}

func TestInit_Success(t *testing.T) {
	workDir := util.GetGitRepoRootDir()
	err := Init(workDir)
	assert.NoError(t, err)
	assert.Equal(t, false, defaultConfig.debugMode)
}

func TestInit_Debug(t *testing.T) {
	os.Setenv("DEBUG", "true")
	workDir := util.GetGitRepoRootDir()
	err := Init(workDir)
	assert.NoError(t, err)
	assert.Equal(t, true, defaultConfig.debugMode)
	assert.True(t, IsDebug())
}

func TestConfig_GetTemplate(t *testing.T) {
	defaultConfig = &Config{
		templates: map[string]string{
			"key": "value",
		},
	}

	assert.Equal(t, "value", defaultConfig.GetTemplate("key"))
}
func TestGetMilvusConfigTemplate(t *testing.T) {
	defaultConfig = &Config{
		templates: map[string]string{
			MilvusConfigTpl:        "value",
			MilvusClusterConfigTpl: "value2",
		},
	}
	assert.Equal(t, "value2", GetMilvusClusterConfigTemplate())
	assert.Equal(t, "value", GetMilvusConfigTemplate())
}
