package config

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/spf13/viper"
)

type ProductConfig struct {
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

	Product struct {
		MaxQueryLimit        int      `mapstructure:"max_query_limit"`
		DefaultPageSize      int      `mapstructure:"default_page_size"`
		ImageValidationEnabled bool     `mapstructure:"image_validation_enabled"`
		MaxImageSize         int      `mapstructure:"max_image_size"`
		AllowedImageFormats  []string `mapstructure:"allowed_image_formats"`
	} `mapstructure:"product"`
}

var (
	productConfig     *ProductConfig
	productConfigLock sync.RWMutex
)

func LoadProductConfig(path string) (*ProductConfig, error) {
	viper.SetConfigFile(path)

	err := viper.ReadInConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config ProductConfig
	err = viper.Unmarshal(&config)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	productConfigLock.Lock()
	productConfig = &config
	productConfigLock.Unlock()

	log.Printf("Loaded product config from %s", path)
	return &config, nil
}

func GetProductConfig() *ProductConfig {
	productConfigLock.RLock()
	defer productConfigLock.RUnlock()
	return productConfig
}