package config

import (
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
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	DBName   string `yaml:"dbname"`
	TLS      string `yaml:"tls"`
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
