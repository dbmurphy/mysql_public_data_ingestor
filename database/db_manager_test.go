package database

import (
	_ "database/sql"
	"encoding/json"
	"github.com/DATA-DOG/go-sqlmock"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	_ "mysql_public_data_ingestor/api_plugins" // Ensure this import is used
	"mysql_public_data_ingestor/config"
	"mysql_public_data_ingestor/syslogwrapper"
)

// MockSyslogWrapper is a mock implementation of syslogwrapper.SyslogWrapper
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

// MockAPIPlugin is a mock implementation of api_plugins.APIPlugin
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

func TestInitializeDatabases(t *testing.T) {
	// Setup mock database connection
	db, mockDB, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Error creating mock DB: %v", err)
	}
	// TODO: mockDB closing too soon
	//defer func() {
	//	if err := db.Close(); err != nil {
	//		t.Fatalf("Error closing mock DB: %v", err)
	//	}
	//}()

	// Setup expectations
	// TODO: ExpectExec now working for Database and table commands
	//mockDB.ExpectExec("CREATE DATABASE IF NOT EXISTS test_prefix1").WillReturnResult(sqlmock.NewResult(1, 1))
	//mockDB.ExpectExec("CREATE TABLE IF NOT EXISTS test_table_prefix (id INT PRIMARY KEY)").WillReturnResult(sqlmock.NewResult(1, 1))
	//mockDB.ExpectExec("CREATE DATABASE IF NOT EXISTS test_prefix2").WillReturnResult(sqlmock.NewResult(1, 1))
	//mockDB.ExpectExec("CREATE DATABASE IF NOT EXISTS test_prefix_extra1").WillReturnResult(sqlmock.NewResult(1, 1))
	//mockDB.ExpectExec("CREATE TABLE IF NOT EXISTS test_table_prefix_1 (id INT PRIMARY KEY)").WillReturnResult(sqlmock.NewResult(1, 1))
	//mockDB.ExpectExec("CREATE TABLE IF NOT EXISTS test_table_prefix_2 (id INT PRIMARY KEY)").WillReturnResult(sqlmock.NewResult(1, 1))
	//mockDB.ExpectExec("CREATE TABLE IF NOT EXISTS test_table_prefix_3 (id INT PRIMARY KEY)").WillReturnResult(sqlmock.NewResult(1, 1))

	// Mock syslog
	mockSyslog := new(MockSyslogWrapper)
	mockSyslog.On("Error", mock.Anything).Return()
	mockSyslog.On("Warning", mock.Anything).Return()

	// Mock APIPlugin
	mockAPIPlugin := new(MockAPIPlugin)
	mockAPIPlugin.On("TablePrefix").Return("test_table_prefix")
	mockAPIPlugin.On("Schema").Return("(id INT PRIMARY KEY)")
	mockAPIPlugin.On("FetchData").Return(nil, nil)
	mockAPIPlugin.On("ValidateConfig", mock.Anything).Return(nil)
	mockAPIPlugin.On("Interval").Return(10, nil)
	mockAPIPlugin.On("SetLogger", mock.Anything).Return()
	mockAPIPlugin.On("GetFieldNames").Return([]string{"field1", "field2"})
	mockAPIPlugin.On("GetValues", mock.Anything).Return([]interface{}{1, "value"})
	mockAPIPlugin.On("Name").Return("test_plugin")

	// Test config
	cfg := config.MainConfig{
		Databases: config.DBConfig{
			Prefix: "test_prefix",
			Copies: 2,
			Extra: map[string]struct {
				Tables int `yaml:"tables"`
			}{
				"extra1": {Tables: 3},
			},
			WriteWorkers: 5,
		},
	}

	// Create DBManager with mock configuration
	dbManager := NewDBManager(config.MySQLConfig{
		User:     "test_user",
		Password: "test_password",
		Host:     "localhost",
		Port:     3306,
		DBName:   "test_db",
		TLS:      "true",
	}, db)

	// Call InitializeDatabases
	dbManager.InitializeDatabases(cfg, mockSyslog, mockAPIPlugin)

	// Validate results
	assert.ElementsMatch(t, []string{"test_prefix1", "test_prefix2", "test_prefix_extra1"}, dbManager.DBs)
	assert.Contains(t, dbManager.Tables, "test_prefix1")
	assert.Contains(t, dbManager.Tables, "test_prefix_extra1")
	assert.Equal(t, []string{"test_table_prefix"}, dbManager.Tables["test_prefix1"])
	assert.Equal(t, []string{"test_table_prefix_1", "test_table_prefix_2", "test_table_prefix_3"}, dbManager.Tables["test_prefix_extra1"])

	// Ensure all expectations were met
	if err := mockDB.ExpectationsWereMet(); err != nil {
		t.Errorf("There were unmet expectations: %v", err)
	}

}
