package config

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"mysql_public_data_ingestor/syslogwrapper"
	"os"
	"testing"
)

// MockSyslogWrapper is a mock implementation of syslogwrapper.SyslogWrapper
type MockSyslogWrapper struct {
	mock.Mock
}

// Implement methods of syslogwrapper.SyslogWrapper
func (m *MockSyslogWrapper) Close() {
	m.Called()
}

func (m *MockSyslogWrapper) Warning(message string) {
	m.Called(message)
}

func (m *MockSyslogWrapper) Error(message string) {
	m.Called(message)
}

func (m *MockSyslogWrapper) Info(message string) {
	m.Called(message)
}

func (m *MockSyslogWrapper) Debug(message string) {
	m.Called(message)
}

// Ensure MockSyslogWrapper implements the SyslogWrapper interface
var _ syslogwrapper.SyslogWrapperInterface = (*MockSyslogWrapper)(nil)

// TestLoadConfig_Success tests the successful loading of a configuration file
func TestLoadConfig_Success(t *testing.T) {
	// Create a temporary YAML file for testing
	tempFile, err := os.CreateTemp("", "tmp_config_test.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer func() {
		if err := os.Remove(tempFile.Name()); err != nil {
			t.Fatalf("Failed to remove temp file: %v", err)
		}
	}()

	// Write test data to the file
	configData := `
plugin_spec:
  name: test_plugin
  config:
    auth:
      user: "your_username"
      pass: "your_password"
    interval: 60
    fetch_workers: 1

databases:
  prefix: "test_prefix"
  copies: 3
  extra:
    foo:
      tables: 5
  write_workers: 10

mysql:
  user: "test_user"
  password: "test_password"
  host: "localhost"
  port: 3306
  dbname: "test_db"
  tls_config:
    ca_file: ""
  connection_pool:
    max_open_conns: 0 # use default
    max_idle_conns: 30 # override
    conn_max_lifetime: 0 # use default
`
	if _, err := tempFile.WriteString(configData); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}

	// Reset the file pointer to the beginning
	if _, err := tempFile.Seek(0, 0); err != nil {
		t.Fatalf("Failed to seek to beginning of temp file: %v", err)
	}

	// Create a mock syslog wrapper
	mockSyslog := new(MockSyslogWrapper)
	mockSyslog.On("Error", mock.Anything).Return()

	// Call the function
	config, err := LoadConfig(tempFile.Name(), mockSyslog)
	assert.NoError(t, err, "Error should be nil when loading config")

	// Verify the actual structure of PluginSpec and other fields
	assert.Equal(t, "test_plugin", config.PluginSpec.Name, "Plugin name should match")
	assert.Equal(t, "test_prefix", config.Databases.Prefix, "Database prefix should match")
	assert.Equal(t, 3, config.Databases.Copies, "Database copies should match")
	assert.Equal(t, 5, config.Databases.Extra["foo"].Tables, "Extra tables should match")
	assert.Equal(t, 10, config.Databases.WriteWorkers, "Database write workers should match")
	assert.Equal(t, "test_user", config.MySQL.User, "MySQL user should match")
	assert.Equal(t, "test_password", config.MySQL.Password, "MySQL password should match")
	assert.Equal(t, "localhost", config.MySQL.Host, "MySQL host should match")
	assert.Equal(t, 3306, config.MySQL.Port, "MySQL port should match")
	assert.Equal(t, "test_db", config.MySQL.DBName, "MySQL DBName should match")
	assert.Equal(t, "", config.MySQL.TLSConfig.CAFile, "MySQL CAFile should match")
	assert.Equal(t, 25, config.MySQL.ConnectionPool.MaxOpenConns, "MySQL MaxOpenConns should use default")
	assert.Equal(t, 30, config.MySQL.ConnectionPool.MaxIdleConns, "MySQL MaxIdleConns should be overridden")
	assert.Equal(t, 3600, config.MySQL.ConnectionPool.ConnMaxLifetime, "MySQL ConnMaxLifetime should use default")
}

// TestLoadConfig_FileReadError tests loading configuration from a non-existent file
func TestLoadConfig_FileReadError(t *testing.T) {
	// Create a mock syslog wrapper
	mockSyslog := new(MockSyslogWrapper)
	mockSyslog.On("Error", mock.Anything).Return()

	// Call the function with a non-existent file
	_, err := LoadConfig("non_existent_file.yaml", mockSyslog)
	assert.Error(t, err, "Error should be returned when reading a non-existent file")
	mockSyslog.AssertCalled(t, "Error", mock.MatchedBy(func(msg string) bool {
		return msg != ""
	}))
}

// TestLoadConfig_UnmarshalError tests loading configuration from a file with invalid YAML data
func TestLoadConfig_UnmarshalError(t *testing.T) {
	// Create a temporary YAML file for testing with invalid data
	tempFile, err := os.CreateTemp("", "config_test.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer func() {
		if err := os.Remove(tempFile.Name()); err != nil {
			t.Fatalf("Failed to remove temp file: %v", err)
		}
	}()

	// Write invalid YAML data to the file
	invalidConfigData := `
plugin_spec:
  plugin_name: test_plugin
databases:
  prefix: test_prefix
  copies: invalid_value
`
	if _, err := tempFile.WriteString(invalidConfigData); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}

	// Reset the file pointer to the beginning
	if _, err := tempFile.Seek(0, 0); err != nil {
		t.Fatalf("Failed to seek to beginning of temp file: %v", err)
	}

	// Create a mock syslog wrapper
	mockSyslog := new(MockSyslogWrapper)
	mockSyslog.On("Error", mock.Anything).Return()

	// Call the function
	_, err = LoadConfig(tempFile.Name(), mockSyslog)
	assert.Error(t, err, "Error should be returned when unmarshalling invalid YAML")
	mockSyslog.AssertCalled(t, "Error", mock.MatchedBy(func(msg string) bool {
		return msg != ""
	}))
}
