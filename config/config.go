package config

import (
	"crypto/tls"
	"fmt"
	"gopkg.in/yaml.v2"
	"mysql_public_data_ingestor/api_plugins"
	"mysql_public_data_ingestor/syslogwrapper"
	"os"
	"reflect"
	"time"
)

type DBConfig struct {
	Prefix string `yaml:"prefix"`
	Copies int    `yaml:"copies"`
	Extra  map[string]struct {
		Tables int `yaml:"tables"`
	} `yaml:"extra"`
	WriteWorkers int `yaml:"write_workers"`
}

type MySQLConfig struct {
	User           string         `yaml:"user"`
	Password       string         `yaml:"password"`
	Host           string         `yaml:"host"`
	Port           int            `yaml:"port"`
	DBName         string         `yaml:"dbname"`
	TLSConfig      TLSConfig      `yaml:"tls_config"`
	ConnectionPool ConnectionPool `yaml:"connection_pool"`
}

// TLSConfig holds the TLS configuration options
type TLSConfig struct {
	CAFile             string
	CertFile           string
	KeyFile            string
	InsecureSkipVerify bool
	ServerName         string
	MinVersion         uint16
	MaxVersion         uint16
	CipherSuites       []uint16
	ClientAuth         tls.ClientAuthType
}

// ConnectionPool holds the connection pool configuration
type ConnectionPool struct {
	MaxOpenConns    int `yaml:"max_open_conns"`
	MaxIdleConns    int `yaml:"max_idle_conns"`
	ConnMaxLifetime int `yaml:"conn_max_lifetime"` // in seconds
}

// NewConnectionPool initializes ConnectionPool with default values
func NewConnectionPool() ConnectionPool {
	return ConnectionPool{
		MaxOpenConns:    25,
		MaxIdleConns:    25,
		ConnMaxLifetime: int(time.Hour.Seconds()), // 3600 seconds
	}
}

type MainConfig struct {
	PluginSpec api_plugins.PluginSpec `yaml:"plugin_spec"`
	Databases  DBConfig               `yaml:"databases"`
	MySQL      MySQLConfig            `yaml:"mysql"`
}

// ValidateConnectionPool ensures the ConnectionPool has default values if they are not provided
func ValidateConnectionPool(config *MainConfig) {
	// Create a struct with default values
	connectionPoolDefaults := NewConnectionPool()

	poolConfigDefaults := reflect.ValueOf(connectionPoolDefaults)
	poolConfigValues := reflect.ValueOf(&config.MySQL.ConnectionPool).Elem()
	configType := poolConfigValues.Type()

	// Iterate through the fields of the ConnectionPool struct by name
	for i := 0; i < poolConfigValues.NumField(); i++ {
		configValue := poolConfigValues.Field(i)
		configKey := configType.Field(i).Name
		defaultValue := poolConfigDefaults.FieldByName(configKey)

		// Check if the field is an integer and its value is zero
		if configValue.Kind() != reflect.Int || configValue.Int() == 0 {
			configValue.Set(defaultValue)
		}
	}
}

// LoadConfig loads the configuration from a file and overrides defaults
func LoadConfig(filename string, sysLog syslogwrapper.SyslogWrapperInterface) (MainConfig, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		sysLog.Error(fmt.Sprintf("Failed to read config file: %v", err))
		return MainConfig{}, err
	}

	var config MainConfig
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		sysLog.Error(fmt.Sprintf("Failed to unmarshal config file: %v", err))
		return MainConfig{}, err
	}

	ValidateConnectionPool(&config)

	return config, nil
}
