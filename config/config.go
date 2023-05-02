package config

import (
	"model-hub/models"
	"os"

	"gopkg.in/yaml.v2"
)

type Model struct {
	Name    models.ModelName `yaml:"name"`
	Path    string           `yaml:"path"`
	Handler string           `yaml:"handler"`
	Workers int              `yaml:"workers"`
}

type Config struct {
	Models map[string]Model `yaml:"models"`
}

func Load(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	cfg := &Config{}
	err = yaml.Unmarshal(data, cfg)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}
