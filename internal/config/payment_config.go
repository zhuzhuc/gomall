package config

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/spf13/viper"
)

type PaymentConfig struct {
	ServiceName    string `mapstructure:"service_name"`
	ServiceVersion string `mapstructure:"service_version"`

	Database struct {
		Host                 string        `mapstructure:"host"`
		Port                 int           `mapstructure:"port"`
		Name                 string        `mapstructure:"name"`
		User                 string        `mapstructure:"user"`
		Password             string        `mapstructure:"password"`
		MaxOpenConnections   int           `mapstructure:"max_open_connections"`
		MaxIdleConnections   int           `mapstructure:"max_idle_connections"`
		ConnectionMaxLifetime time.Duration `mapstructure:"connection_max_lifetime"`
	} `mapstructure:"database"`

	Registration struct {
		Etcd struct {
			Endpoints   []string      `mapstructure:"endpoints"`
			DialTimeout time.Duration `mapstructure:"dial_timeout"`
		} `mapstructure:"etcd"`
	} `mapstructure:"registration"`

	Payment struct {
		DefaultPageSize   int `mapstructure:"default_page_size"`
		MaxQueryLimit     int `mapstructure:"max_query_limit"`
		TransactionTimeout time.Duration `mapstructure:"transaction_timeout"`
		PricePrecision    int `mapstructure:"price_precision"`
	} `mapstructure:"payment"`
}

var (
	paymentConfig     *PaymentConfig
	paymentConfigLock sync.RWMutex
)

func LoadPaymentConfig(path string) (*PaymentConfig, error) {
	viper.SetConfigFile(path)
	viper.SetConfigType("yaml")

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config PaymentConfig
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	paymentConfigLock.Lock()
	defer paymentConfigLock.Unlock()
	paymentConfig = &config

	log.Printf("Loaded payment config from %s", path)
	return &config, nil
}

func GetPaymentConfig() *PaymentConfig {
	paymentConfigLock.RLock()
	defer paymentConfigLock.RUnlock()
	return paymentConfig
}