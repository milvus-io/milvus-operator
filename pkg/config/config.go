package config

import (
	"io/ioutil"
	"os"

	corev1 "k8s.io/api/core/v1"
)

const (
	DefaultMilvusVersion   = "v2.0.0-rc5-hotfix1-20210901-9e0b2cc"
	DefaultMilvusBaseImage = "milvusdb/milvus"
	DefaultMilvusImage     = DefaultMilvusBaseImage + ":" + DefaultMilvusVersion
	DefaultImagePullPolicy = corev1.PullIfNotPresent
)

const (
	TemplateDir  = "config/assets/templates"
	ChartDir     = "config/assets/charts"
	ProviderName = "milvus-operator"
)

var (
	defaultConfig *Config
)

func Init() error {
	c, err := NewConfig()
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

func GetTemplate(name string) string {
	return defaultConfig.templates[name]
}

type Config struct {
	debugMode bool
	certDir   string
	templates map[string]string
}

func NewConfig() (*Config, error) {
	config := &Config{
		templates: make(map[string]string),
	}

	tmpls, err := ioutil.ReadDir(TemplateDir)
	if err != nil {
		return nil, err
	}
	for _, tmpl := range tmpls {
		data, err := ioutil.ReadFile(TemplateDir + "/" + tmpl.Name())
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
