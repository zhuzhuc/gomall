package config

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/spf13/viper"
)

type OrderConfig struct {
	ServiceName    string `mapstructure:"service_name"`
	ServiceVersion string `mapstructure:"service_version"`

	Database struct {
		Host     string `mapstructure:"host"`
		Port     int    `mapstructure:"port"`
		Name     string `mapstructure:"name"`
		User     string `mapstructure:"user"`
		Password string `mapstructure:"password"`
	} `mapstructure:"database"`

	Registration struct {
		Etcd struct {
			Endpoints   []string      `mapstructure:"endpoints"`
			DialTimeout time.Duration `mapstructure:"dial_timeout"`
		} `mapstructure:"etcd"`
	} `mapstructure:"registration"`

	Order struct {
		DefaultPageSize   int `mapstructure:"default_page_size"`
		MaxQueryLimit     int `mapstructure:"max_query_limit"`
		AutoCancelMinutes int `mapstructure:"auto_cancel_minutes"`
		PricePrecision    int `mapstructure:"price_precision"`
	} `mapstructure:"order"`
}

var (
	orderConfig     *OrderConfig
	orderConfigLock sync.RWMutex
)

func LoadOrderConfig(path string) (*OrderConfig, error) {
	viper.SetConfigFile(path)

	err := viper.ReadInConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config OrderConfig
	err = viper.Unmarshal(&config)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	orderConfigLock.Lock()
	orderConfig = &config
	orderConfigLock.Unlock()

	log.Printf("Loaded order config from %s", path)
	return &config, nil
}

func GetOrderConfig() *OrderConfig {
	orderConfigLock.RLock()
	defer orderConfigLock.RUnlock()
	return orderConfig
}