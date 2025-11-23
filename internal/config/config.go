package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

// Config 应用配置
type Config struct {
	Env      string         `mapstructure:"env"` // 环境: development, production
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	OpenFGA  OpenFGAConfig  `mapstructure:"openfga"`
	Keycloak KeycloakConfig `mapstructure:"keycloak"`
	CORS     CORSConfig     `mapstructure:"cors"`
	Log      LogConfig      `mapstructure:"log"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Host            string `mapstructure:"host"`
	Port            int    `mapstructure:"port"`
	User            string `mapstructure:"user"`
	Password        string `mapstructure:"password"`
	DBName          string `mapstructure:"dbname"`
	SSLMode         string `mapstructure:"sslmode"`
	MaxIdleConns    int    `mapstructure:"max_idle_conns"`
	MaxOpenConns    int    `mapstructure:"max_open_conns"`
	ConnMaxLifetime int    `mapstructure:"conn_max_lifetime"` // 秒
	ConnMaxIdleTime int    `mapstructure:"conn_max_idle_time"` // 秒
}

// OpenFGAConfig OpenFGA 配置
type OpenFGAConfig struct {
	APIURL  string `mapstructure:"api_url"`
	StoreID string `mapstructure:"store_id"`
	ModelID string `mapstructure:"model_id"`
}

// KeycloakConfig Keycloak 配置
type KeycloakConfig struct {
	Issuer  string `mapstructure:"issuer"`
	JWKSURL string `mapstructure:"jwks_url"`
}

// CORSConfig CORS 配置
type CORSConfig struct {
	AllowedOrigins []string `mapstructure:"allowed_origins"`
	AllowedMethods []string `mapstructure:"allowed_methods"`
	AllowedHeaders []string `mapstructure:"allowed_headers"`
	MaxAge         int      `mapstructure:"max_age"`
}

// LogConfig 日志配置
type LogConfig struct {
	Level  string `mapstructure:"level"`  // 日志级别: debug, info, warn, error
	Format string `mapstructure:"format"`  // 日志格式: json, text
	Output string `mapstructure:"output"` // 输出位置: stdout, file, both
}

// Load 加载配置,支持配置文件和环境变量
func Load(configPath string) (*Config, error) {
	v := viper.New()
	
	// 设置默认值
	setDefaults(v)
	
	// 如果提供了配置文件路径,从文件加载
	if configPath != "" {
		v.SetConfigFile(configPath)
		if err := v.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	} else {
		// 尝试从默认位置加载
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		v.AddConfigPath("./config")
		v.AddConfigPath("$HOME/.approval-gin")
		// 忽略配置文件不存在的错误,使用默认值
		_ = v.ReadInConfig()
	}
	
	// 支持环境变量
	v.SetEnvPrefix("APP")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	
	return &cfg, nil
}

// IsProduction 判断是否为生产环境
func IsProduction(cfg *Config) bool {
	if cfg == nil {
		return false
	}
	return cfg.Env == "production"
}

// Default 返回默认配置
func Default() *Config {
	v := viper.New()
	setDefaults(v)
	
	var cfg Config
	_ = v.Unmarshal(&cfg)
	return &cfg
}

// setDefaults 设置配置默认值
func setDefaults(v *viper.Viper) {
	// 环境变量
	env := v.GetString("env")
	if env == "" {
		env = os.Getenv("APP_ENV")
		if env == "" {
			env = "development"
		}
	}
	v.SetDefault("env", env)

	// 服务器默认配置
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8080)
	
	// 数据库默认配置
	v.SetDefault("database.host", "localhost")
	v.SetDefault("database.port", 5432)
	v.SetDefault("database.user", "postgres")
	v.SetDefault("database.password", "")
	v.SetDefault("database.dbname", "approval")
	v.SetDefault("database.sslmode", "disable")
	
	// 数据库连接池配置（根据环境设置默认值）
	if env == "production" {
		v.SetDefault("database.max_idle_conns", 20)
		v.SetDefault("database.max_open_conns", 200)
		v.SetDefault("database.conn_max_lifetime", 3600) // 1 小时
		v.SetDefault("database.conn_max_idle_time", 300)  // 5 分钟
	} else {
		v.SetDefault("database.max_idle_conns", 10)
		v.SetDefault("database.max_open_conns", 100)
		v.SetDefault("database.conn_max_lifetime", 3600) // 1 小时
		v.SetDefault("database.conn_max_idle_time", 600)  // 10 分钟
	}
	
	// OpenFGA 默认配置
	v.SetDefault("openfga.api_url", "http://localhost:8081")
	v.SetDefault("openfga.store_id", "")
	v.SetDefault("openfga.model_id", "")
	
	// Keycloak 默认配置
	v.SetDefault("keycloak.issuer", "")
	v.SetDefault("keycloak.jwks_url", "")
	
	// CORS 默认配置
	v.SetDefault("cors.allowed_origins", []string{"*"})
	v.SetDefault("cors.allowed_methods", []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"})
	v.SetDefault("cors.allowed_headers", []string{"Content-Type", "Authorization", "X-Request-ID"})
	v.SetDefault("cors.max_age", 86400)
	
	// 日志配置（根据环境设置默认值）
	if env == "production" {
		v.SetDefault("log.level", "warn")
		v.SetDefault("log.format", "json")
	} else {
		v.SetDefault("log.level", "debug")
		v.SetDefault("log.format", "text")
	}
	v.SetDefault("log.output", "stdout")
}

