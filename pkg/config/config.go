package config

import (
	"io/ioutil"
	"os"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
)

var (
	version = "0.2.5" // inject when go build through ldflag
	commit  = ""      // inject when go build through ldflag
)

const (
	VersionLabel = "milvus.io/operator-version"

	DefaultMilvusVersion   = "v2.0.0-pre-ga"
	DefaultMilvusBaseImage = "milvusdb/milvus"
	DefaultMilvusImage     = DefaultMilvusBaseImage + ":" + DefaultMilvusVersion
	DefaultImagePullPolicy = corev1.PullIfNotPresent
	MilvusConfigTpl        = "milvus.yaml.tmpl"
	MilvusClusterConfigTpl = "milvus-cluster.yaml.tmpl"
)

const (
	TemplateRelativeDir = "config/assets/templates"
	ChartDir            = "config/assets/charts"
	ProviderName        = "milvus-operator"
)

// GetVersion returns the version of the operator
func GetVersion() string {
	return version
}

// GetCommit returns the 8byte git commit id of the operator
func GetCommit() string {
	return commit
}

func PrintVersionMessage(logger logr.Logger) {
	logger.WithValues("version", version, "commit", commit, "default milvus", DefaultMilvusVersion).Info("Operator Version Info")
}

var (
	defaultConfig *Config
)

func Init(workDir string) error {
	c, err := NewConfig(workDir)
	if err != nil {
		return err
	}
	defaultConfig = c
	if os.Getenv("DEBUG") == "true" {
		defaultConfig.debugMode = true
	}

	return nil
}

func IsDebug() bool {
	return defaultConfig.debugMode
}

func GetMilvusConfigTemplate() string {
	return defaultConfig.GetTemplate(MilvusConfigTpl)
}

func GetMilvusClusterConfigTemplate() string {
	return defaultConfig.GetTemplate(MilvusClusterConfigTpl)
}

type Config struct {
	debugMode bool
	templates map[string]string
}

func NewConfig(workDir string) (*Config, error) {
	config := &Config{
		templates: make(map[string]string),
	}

	templateDir := workDir + TemplateRelativeDir

	tmpls, err := ioutil.ReadDir(templateDir)
	if err != nil {
		return nil, err
	}
	for _, tmpl := range tmpls {
		data, err := ioutil.ReadFile(templateDir + "/" + tmpl.Name())
		if err != nil {
			return nil, err
		}

		config.templates[tmpl.Name()] = string(data)
	}

	return config, nil
}

func (c Config) GetTemplate(name string) string {
	return c.templates[name]
}
