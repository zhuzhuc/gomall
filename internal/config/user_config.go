package config

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/spf13/viper"
)

type UserConfig struct {
	ServiceName    string `mapstructure:"service_name"`
	ServiceVersion string `mapstructure:"service_version"`

	Database struct {
		Driver                string        `mapstructure:"driver"`
		Host                  string        `mapstructure:"host"`
		Port                  int           `mapstructure:"port"`
		Name                  string        `mapstructure:"name"`
		User                  string        `mapstructure:"user"`
		Password              string        `mapstructure:"password"`
		MaxOpenConnections    int           `mapstructure:"max_open_connections"`
		MaxIdleConnections    int           `mapstructure:"max_idle_connections"`
		ConnectionMaxLifetime time.Duration `mapstructure:"connection_max_lifetime"`
	} `mapstructure:"database"`

	Password struct {
		Salt           string `mapstructure:"salt"`
		HashIterations int    `mapstructure:"hash_iterations"`
	} `mapstructure:"password"`

	Registration struct {
		Etcd struct {
			Endpoints   []string      `mapstructure:"endpoints"`
			DialTimeout time.Duration `mapstructure:"dial_timeout"`
		} `mapstructure:"etcd"`
	} `mapstructure:"registration"`

	Security struct {
		MaxLoginAttempts          int           `mapstructure:"max_login_attempts"`
		LoginAttemptResetDuration time.Duration `mapstructure:"login_attempt_reset_duration"`
		PasswordMinLength         int           `mapstructure:"password_min_length"`
		PasswordComplexityEnabled bool          `mapstructure:"password_complexity_enabled"`
	} `mapstructure:"security"`
}

var (
	userConfig     *UserConfig
	userConfigLock sync.RWMutex
)

func LoadUserConfig(configPath string) (*UserConfig, error) {
	// 初始化 Viper
	v := viper.New()
	
	// 设置配置文件类型
	v.SetConfigType("yaml")
	
	// 设置配置文件路径
	v.SetConfigFile(configPath)
	
	// 允许环境变量覆盖配置
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	
	// 读取配置文件
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	// 绑定配置到结构体
	var config UserConfig
	
	// 使用 Viper 的 Unmarshal 方法
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("unable to decode into config struct: %w", err)
	}
	
	// 如果某些关键配置为空，尝试从环境变量获取
	if config.Database.Host == "" {
		config.Database.Host = v.GetString("database.host")
	}
	if config.Database.Port == 0 {
		config.Database.Port = v.GetInt("database.port")
	}
	if config.Database.Name == "" {
		config.Database.Name = v.GetString("database.name")
	}
	if config.Database.User == "" {
		config.Database.User = v.GetString("database.user")
	}
	if config.Database.Password == "" {
		config.Database.Password = v.GetString("database.password")
	}
	
	log.Printf("Loaded user configuration from %s", configPath)
	userConfigLock.Lock()
	defer userConfigLock.Unlock()
	userConfig = &config
	
	return &config, nil
}

func GetUserConfig() *UserConfig {
	userConfigLock.RLock()
	defer userConfigLock.RUnlock()
	return userConfig
}
