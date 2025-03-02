package config

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/spf13/viper"
)

type AuthConfig struct {
	JWT struct {
		SecretKey string        `mapstructure:"secret_key"`
		TokenTTL  time.Duration `mapstructure:"token_ttl"`
		Issuer    string        `mapstructure:"issuer"`
	} `mapstructure:"jwt"`

	Security struct {
		TokenBlacklistMaxSize       int           `mapstructure:"token_blacklist_max_size"`
		TokenBlacklistCleanupPeriod time.Duration `mapstructure:"token_blacklist_cleanup_interval"`
	} `mapstructure:"security"`

	Registration struct {
		ServiceName    string `mapstructure:"service_name"`
		ServiceVersion string `mapstructure:"service_version"`
		Etcd           struct {
			Endpoints   []string      `mapstructure:"endpoints"`
			DialTimeout time.Duration `mapstructure:"dial_timeout"`
		} `mapstructure:"etcd"`
	} `mapstructure:"registration"`
}

var (
	config     *AuthConfig
	configLock sync.RWMutex
)

func LoadConfig(path string) (*AuthConfig, error) {
	viper.SetConfigFile(path)
	viper.SetConfigType("yaml")

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	var cfg AuthConfig
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	configLock.Lock()
	defer configLock.Unlock()
	config = &cfg

	log.Printf("Loaded configuration from %s", path)
	return &cfg, nil
}

func GetConfig() *AuthConfig {
	configLock.RLock()
	defer configLock.RUnlock()
	return config
}
