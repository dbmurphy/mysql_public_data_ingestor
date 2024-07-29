package mysql_public_data_ingestor

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"mysql_public_data_ingestor/api_plugins"
	"mysql_public_data_ingestor/syslogwrapper"
	"os"
	"testing"
)

// Mock implementations for testing
type MockSyslogWrapper struct {
	mock.Mock
}

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

type MockAPIPlugin struct {
	mock.Mock
}

func (m *MockAPIPlugin) TablePrefix() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockAPIPlugin) Schema() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockAPIPlugin) FetchData() (interface{}, error) {
	args := m.Called()
	return args.Get(0), args.Error(1)
}

func (m *MockAPIPlugin) ValidateConfig(config json.RawMessage) error {
	args := m.Called(config)
	return args.Error(0)
}

func (m *MockAPIPlugin) Interval() (int, error) {
	args := m.Called()
	return args.Int(0), args.Error(1)
}

func (m *MockAPIPlugin) SetLogger(sysLog syslogwrapper.SyslogWrapperInterface) {
	m.Called(sysLog)
}

func (m *MockAPIPlugin) GetFieldNames() []string {
	args := m.Called()
	return args.Get(0).([]string)
}

func (m *MockAPIPlugin) GetValues(record interface{}) []interface{} {
	args := m.Called(record)
	return args.Get(0).([]interface{})
}

func (m *MockAPIPlugin) Name() string {
	args := m.Called()
	return args.String(0)
}

// Test for FetchAndDistributeData function
func TestFetchAndDistributeData(t *testing.T) {
	// Mock Syslog
	mockSyslog := new(MockSyslogWrapper)

	// Mock APIPlugin
	mockAPIPlugin := new(MockAPIPlugin)
	mockAPIPlugin.On("FetchData").Return(api_plugins.Response{Records: []interface{}{"record1", "record2"}}, nil)
	mockAPIPlugin.On("GetFieldNames").Return([]string{"field1", "field2"})
	mockAPIPlugin.On("GetValues", mock.Anything).Return([]interface{}{1, "value"})
	t.Logf("Setup Mock and catch methods...")
	tableChannels := make(map[string]chan []interface{})
	tableChannels["db.table"] = make(chan []interface{})
	t.Logf("Setup tableChannels...")

	err := FetchAndDistributeData(mockAPIPlugin, tableChannels, mockSyslog)
	assert.NoError(t, err)
	t.Logf("Ran FetchAndDistributeData...")

	// Check channel data
	batchData := <-tableChannels["db.table"]
	assert.Equal(t, 2, len(batchData))
}

// Test for TableWorker function
// TODO: Rework TableWorker, so it can be passed a mockDB object vs making a connection with the DSN maybe using db_manager
//func TestTableWorker(t *testing.T) {
//	// Mock Syslog
//	mockSyslog := new(MockSyslogWrapper)
//
//	// Setup mock database connection
//	db, mockDB, err := sqlmock.New()
//	if err != nil {
//		t.Fatalf("Error creating mock DB: %v", err)
//	}
//	defer func() {
//		if err := db.Close(); err != nil {
//			mockSyslog.Warning(fmt.Sprintf("Failed to close MySQL connection: %v", err))
//		}
//	}()
//
//	// Mock APIPlugin
//	mockAPIPlugin := new(MockAPIPlugin)
//	mockAPIPlugin.On("GetFieldNames").Return([]string{"field1", "field2"})
//	mockAPIPlugin.On("GetValues", mock.Anything).Return([]interface{}{1, "value"})
//
//	// Mock the SQL expectations
//	mockDB.ExpectExec("INSERT INTO test_db.test_table").WithArgs(1, "value").WillReturnResult(sqlmock.NewResult(1, 1))
//
//	// Setup table worker
//	var wg sync.WaitGroup
//	batchChan := make(chan []interface{})
//	wg.Add(1)
//
//	go TableWorker("test_db", "test_table", batchChan, &wg, mockSyslog, "test_dsn", mockAPIPlugin)
//
//	// Send test data
//	batchChan <- []interface{}{"record1", "record2"}
//	close(batchChan)
//
//	wg.Wait()
//
//	// Check SQL expectations
//	if err := mockDB.ExpectationsWereMet(); err != nil {
//		t.Errorf("There were unmet expectations: %v", err)
//	}
//}

// Test for SetupSyslog function
func TestSetupSyslog(t *testing.T) {
	mockSyslog, err := SetupSyslog("test_tag")
	if err != nil {
		t.Fatalf("Error setting up syslog: %v", err)
	}

	if mockSyslog == nil {
		t.Fatal("Expected non-nil syslog wrapper")
	}
}

// Test for LoadConfig function
func TestLoadConfig(t *testing.T) {
	mockSyslog := new(MockSyslogWrapper)
	mockSyslog.On("Error", mock.Anything).Return()

	// Set a test config file path
	configPath := "config-test.yaml"
	os.Setenv("TEST_CONFIG_FILE", configPath)
	defer os.Unsetenv("TEST_CONFIG_FILE")

	cfg, err := LoadConfig(mockSyslog)
	if err != nil {
		t.Fatalf("Error loading config: %v", err)
	}

	if cfg.PluginSpec.Name == "" {
		t.Fatal("Expected non-empty plugin name in config")
	}
}

//// Test for SetupPlugins function
//func TestSetupPlugins(t *testing.T) {
//	mockSyslog := new(MockSyslogWrapper)
//	mockSyslog.On("Error", mock.Anything).Return()
//
//	cfg := config.MainConfig{
//		PluginSpec: config.PluginSpec{
//			Name: "test_plugin",
//		},
//	}
//
//	mockAPIPlugin := new(MockAPIPlugin)
//	api_plugins.InitPlugin = func(name string) (api_plugins.APIPlugin, error) {
//		return mockAPIPlugin, nil
//	}
//
//	plugin, err := SetupPlugins(cfg, mockSyslog)
//	if err != nil {
//		t.Fatalf("Error setting up plugins: %v", err)
//	}
//
//	if plugin == nil {
//		t.Fatal("Expected non-nil plugin")
//	}
//}