package api_plugins

import (
	"encoding/json"
	"mysql_public_data_ingestor/syslogwrapper"
)

type PluginSpec struct {
	Name   string                 `yaml:"name"`
	Config map[string]interface{} `yaml:"config"`
}

type Response struct {
	Records []interface{}
}

type APIPlugin interface {
	FetchData() (interface{}, error)
	Schema() string
	TablePrefix() string
	ValidateConfig(config json.RawMessage) error
	Interval() (int, error)
	SetLogger(sysLog syslogwrapper.SyslogWrapperInterface)
	GetFieldNames() []string
	GetValues(record interface{}) []interface{}
	Name() string // Added method
}
