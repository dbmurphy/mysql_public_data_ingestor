package config

import (
	"crypto/tls"
	"fmt"
	"gopkg.in/yaml.v2"
	"mysql_public_data_ingestor/api_plugins"
	"mysql_public_data_ingestor/syslogwrapper"
	"os"
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

type ConnectionPool struct {
	MaxOpenConns    int `yaml:"max_open_conns"`
	MaxIdleConns    int `yaml:"max_idle_conns"`
	ConnMaxLifetime int `yaml:"conn_max_lifetime"` // in seconds
}

type MainConfig struct {
	PluginSpec api_plugins.PluginSpec `yaml:"plugin_spec"`
	Databases  DBConfig               `yaml:"databases"`
	MySQL      MySQLConfig            `yaml:"mysql"`
}

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

	return config, nil
}
