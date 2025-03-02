package config

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/spf13/viper"
)

type CartConfig struct {
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

	Cart struct {
		MaxItemsPerCart   int `mapstructure:"max_items_per_cart"`
		DefaultPageSize   int `mapstructure:"default_page_size"`
		MaxQueryLimit     int `mapstructure:"max_query_limit"`
		ItemQuantityLimit int `mapstructure:"item_quantity_limit"`
		PricePrecision    int `mapstructure:"price_precision"`
	} `mapstructure:"cart"`
}

var (
	cartConfig     *CartConfig
	cartConfigLock sync.RWMutex
)

func LoadCartConfig(path string) (*CartConfig, error) {
	viper.SetConfigFile(path)

	err := viper.ReadInConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config CartConfig
	err = viper.Unmarshal(&config)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	cartConfigLock.Lock()
	cartConfig = &config
	cartConfigLock.Unlock()

	log.Printf("Loaded cart config from %s", path)
	return &config, nil
}

func GetCartConfig() *CartConfig {
	cartConfigLock.RLock()
	defer cartConfigLock.RUnlock()
	return cartConfig
}