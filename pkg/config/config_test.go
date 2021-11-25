package config

import (
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func getGitRepoRootDir() string {
	_, filename, _, _ := runtime.Caller(0)
	return strings.TrimSuffix(filename, "pkg/config/config_test.go")
}

func TestInit_NewConfigFailed(t *testing.T) {
	workDir = "/a-bad-path/a-bad-path"
	err := Init()
	assert.Error(t, err)
}

func TestInit_Success(t *testing.T) {
	workDir = getGitRepoRootDir()
	err := Init()
	assert.NoError(t, err)
	assert.Equal(t, false, defaultConfig.debugMode)
}

func TestInit_Debug(t *testing.T) {
	os.Setenv("DEBUG", "true")
	workDir = getGitRepoRootDir()
	err := Init()
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
			MilvusConfigTpl: "value",
		},
	}
	assert.Equal(t, "value", GetMilvusConfigTemplate())
}
