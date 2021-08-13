package config

import (
	"io/ioutil"
)

const (
	TemplateDir     = "config/assets"
	ProviderName    = "milvus-operator"
	DefaultPriority = 5
)

type Config struct {
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
