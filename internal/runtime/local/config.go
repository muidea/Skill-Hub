package local

import (
	"github.com/muidea/skill-hub/internal/config"
	"github.com/muidea/skill-hub/internal/pkg/configport"
)

type Config struct{}

var _ configport.Config = (*Config)(nil)

func NewConfig() *Config                   { return &Config{} }
func (c *Config) RootDir() (string, error) { return config.GetRootDir() }
func (c *Config) Initialized() error       { _, err := config.GetConfig(); return err }
