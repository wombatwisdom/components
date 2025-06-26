package spec

import (
	"strings"

	mapstructure "github.com/go-viper/mapstructure/v2"
	"gopkg.in/yaml.v3"
)

func NewYamlConfig(raw string, replacements ...string) Config {
	cfg := strings.TrimSpace(raw)
	if len(replacements) > 0 {
		cfg = strings.NewReplacer(replacements...).Replace(cfg)
	}
	return &yamlConfig{
		raw: []byte(cfg),
	}
}

func NewMapConfig(raw map[string]any) Config {
	return &mapConfig{
		raw: raw,
	}
}

type Config interface {
	Decode(target any) error
}

type yamlConfig struct {
	raw []byte
}

func (c *yamlConfig) Decode(target any) error {
	return yaml.Unmarshal(c.raw, target)
}

type mapConfig struct {
	raw map[string]any
}

func (c *mapConfig) Decode(target any) error {
	return mapstructure.Decode(c.raw, target)
}
