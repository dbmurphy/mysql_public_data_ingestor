package opensky

import (
	"encoding/json"
	"errors"
	"fmt"
	"mysql_public_data_ingestor/syslogwrapper"
	"net/http"
	"strings"
)

type Auth struct {
	User string `json:"user"`
	Pass string `json:"pass"`
}

type Config struct {
	Auth         Auth `json:"auth"`
	Interval     int  `json:"interval"`
	FetchWorkers int  `json:"fetch_workers"`
}

type SkyResponse struct {
	Time   int64           `json:"time"`
	States [][]interface{} `json:"states"`
}

type Plugin struct {
	Config       Config
	sysLog       syslogwrapper.SyslogWrapperInterface
	FetchDataURL string
}

// Centralized schema definition
var schema = map[string]string{
	"time":            "INT",
	"icao24":          "VARCHAR(10)",
	"callsign":        "VARCHAR(10)",
	"origin_country":  "VARCHAR(50)",
	"time_position":   "INT",
	"last_contact":    "INT",
	"longitude":       "FLOAT",
	"latitude":        "FLOAT",
	"baro_altitude":   "FLOAT",
	"on_ground":       "BOOLEAN",
	"velocity":        "FLOAT",
	"true_track":      "FLOAT",
	"vertical_rate":   "FLOAT",
	"sensors":         "JSON",
	"geo_altitude":    "FLOAT",
	"squawk":          "VARCHAR(10)",
	"spi":             "BOOLEAN",
	"position_source": "INT",
}

func (p *Plugin) SetLogger(sysLog syslogwrapper.SyslogWrapperInterface) {
	p.sysLog = sysLog
}

func (p *Plugin) ValidateCredentials() error {
	url := p.FetchDataURL
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}
	req.SetBasicAuth(p.Config.Auth.User, p.Config.Auth.Pass)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to validate credentials: %w", err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			p.sysLog.Warning(fmt.Sprintf("Failed to close response body: %v", cerr))
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("invalid credentials, status code: %d", resp.StatusCode)
	}

	return nil
}

func (p *Plugin) FetchData() (interface{}, error) {
	url := p.FetchDataURL
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		p.sysLog.Error(fmt.Sprintf("Failed to create HTTP request: %v", err))
		return SkyResponse{}, fmt.Errorf("failed to create HTTP request: %w", err)
	}
	req.SetBasicAuth(p.Config.Auth.User, p.Config.Auth.Pass)

	resp, err := client.Do(req)
	if err != nil {
		p.sysLog.Error(fmt.Sprintf("Failed to fetch data: %v", err))
		return SkyResponse{}, fmt.Errorf("failed to fetch data: %w", err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			p.sysLog.Warning(fmt.Sprintf("Failed to close response body: %v", cerr))
		}
	}()

	var data SkyResponse
	err = json.NewDecoder(resp.Body).Decode(&data)
	if err != nil {
		p.sysLog.Error(fmt.Sprintf("Failed to decode response: %v", err))
		return SkyResponse{}, fmt.Errorf("failed to decode response: %w", err)
	}

	return data, nil
}

func (p *Plugin) Schema() string {
	var schemaParts []string
	for field, fieldType := range schema {
		schemaParts = append(schemaParts, fmt.Sprintf("%s %s", field, fieldType))
	}
	return fmt.Sprintf("(%s)", strings.Join(schemaParts, ", "))
}

func (p *Plugin) TablePrefix() string {
	return "flights"
}

func (p *Plugin) ValidateConfig(config json.RawMessage) error {
	var skyConfig Config
	err := json.Unmarshal(config, &skyConfig)
	if err != nil {
		p.sysLog.Error(fmt.Sprintf("Invalid config format: %v", err))
		return errors.New("invalid config format")
	}
	if skyConfig.Auth.User == "" || skyConfig.Auth.Pass == "" {
		err = errors.New("missing user or pass in auth")
		p.sysLog.Error(err.Error())
		return err
	}
	p.Config = skyConfig
	p.FetchDataURL = "https://opensky-network.org/api/states/all" // Set the default URL

	// Validate credentials during startup
	if err := p.ValidateCredentials(); err != nil {
		p.sysLog.Error(fmt.Sprintf("Credential validation failed: %v", err))
		return err
	}

	return nil
}

func (p *Plugin) Interval() (int, error) {
	return p.Config.Interval, nil
}

func (p *Plugin) GetFieldNames() []string {
	fieldNames := make([]string, 0, len(schema))
	for field := range schema {
		fieldNames = append(fieldNames, field)
	}
	return fieldNames
}

func (p *Plugin) GetValues(record interface{}) []interface{} {
	values := record.([]interface{})
	return values
}

func (p *Plugin) Name() string {
	return "opensky"
}

// PluginInstance is the exported symbol that will be looked up when loading the plugin.
var PluginInstance Plugin
