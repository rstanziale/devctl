package config

import (
	"fmt"

	"github.com/spf13/viper"
)

func Load(path string) (*Config, string, error) {
	v := viper.New()

	v.SetConfigFile(path)

	if err := v.ReadInConfig(); err != nil {
		return nil, "", fmt.Errorf("cannot read config: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, "", fmt.Errorf("cannot parse config: %w", err)
	}

	return &cfg, v.ConfigFileUsed(), nil
}
