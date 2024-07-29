package api_plugins

import (
	"encoding/json"
	"mysql_public_data_ingestor/syslogwrapper"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock Plugin struct
type MockPlugin struct {
	mock.Mock
}

// Mock Response struct
type MockResponse struct {
	mock.Mock
}

// Implement methods of the Plugin interface or struct as needed
func (m *MockPlugin) Load() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockPlugin) LoadWithError() error {
	args := m.Called()
	return args.Error(0)
}

// Define methods to mock Response methods if needed
func (m *MockResponse) GetTime() int {
	args := m.Called()
	return args.Int(0)
}

func (m *MockResponse) GetStates() [][]interface{} {
	args := m.Called()
	return args.Get(0).([][]interface{})
}

// Example test using the mock
func TestPluginInitialization(t *testing.T) {
	mockPlugin := new(MockPlugin)

	// Setup expectations
	mockPlugin.On("Load").Return(nil)          // Expect Load to return nil (no error)
	mockPlugin.On("LoadWithError").Return(nil) // Expect LoadWithError to return nil (no error)

	// Use the mock in your test
	err := mockPlugin.Load()
	assert.NoError(t, err, "Error should be nil when loading a plugin")

	err = mockPlugin.LoadWithError()
	assert.NoError(t, err, "Error should be nil when loading a plugin with error")
}

// Another test with mock
func TestPluginLoadError(t *testing.T) {
	mockPlugin := new(MockPlugin)

	// Setup expectations
	mockPlugin.On("LoadWithError").Return(assert.AnError) // Expect LoadWithError to return an error

	// Use the mock in your test
	err := mockPlugin.LoadWithError()
	assert.Error(t, err, "Error should be returned when simulating an error")
}

// Test Response mock
func TestResponseMock(t *testing.T) {
	mockResponse := new(MockResponse)

	// Mock data
	mockTime := 1234567890
	mockStates := [][]interface{}{
		{1234567890, "abc123", "CALLSIGN", "Country", 1234567890, 1234567890, 10.0, 20.0, 30.0, true, 40.0, 50.0, 60.0, nil, 70.0, "SQUAWK", true, 1},
	}

	// Setup expectations
	mockResponse.On("GetTime").Return(mockTime)
	mockResponse.On("GetStates").Return(mockStates)

	// Use the mock in your test
	responseTime := mockResponse.GetTime()
	assert.Equal(t, mockTime, responseTime, "Time should match the mocked value")

	states := mockResponse.GetStates()
	assert.Equal(t, mockStates, states, "States should match the mocked values")
}

// Mock APIPlugin implementation
type MockAPIPlugin struct {
	mock.Mock
}

// Implement methods of the APIPlugin interface
func (m *MockAPIPlugin) FetchData() (interface{}, error) {
	args := m.Called()
	return args.Get(0), args.Error(1)
}

func (m *MockAPIPlugin) Schema() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockAPIPlugin) TablePrefix() string {
	args := m.Called()
	return args.String(0)
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

// Test APIPlugin interface implementation
func TestAPIPlugin(t *testing.T) {
	mockPlugin := new(MockAPIPlugin)
	syslogWrapper := new(syslogwrapper.SyslogWrapper)

	// Define mock return values
	mockPlugin.On("FetchData").Return(nil, nil)
	mockPlugin.On("Schema").Return("schema")
	mockPlugin.On("TablePrefix").Return("prefix")
	mockPlugin.On("ValidateConfig", mock.Anything).Return(nil)
	mockPlugin.On("Interval").Return(10, nil)
	mockPlugin.On("GetFieldNames").Return([]string{"field1", "field2"})
	mockPlugin.On("GetValues", mock.Anything).Return([]interface{}{1, "value"})
	mockPlugin.On("Name").Return("TestPlugin")

	// Test FetchData
	data, err := mockPlugin.FetchData()
	assert.NoError(t, err)
	assert.Nil(t, data)

	// Test Schema
	schema := mockPlugin.Schema()
	assert.Equal(t, "schema", schema)

	// Test TablePrefix
	prefix := mockPlugin.TablePrefix()
	assert.Equal(t, "prefix", prefix)

	// Test ValidateConfig
	err = mockPlugin.ValidateConfig(json.RawMessage(`{"key":"value"}`))
	assert.NoError(t, err)

	// Test Interval
	interval, err := mockPlugin.Interval()
	assert.NoError(t, err)
	assert.Equal(t, 10, interval)

	// Test SetLogger
	mockPlugin.On("SetLogger", syslogWrapper).Return()
	mockPlugin.SetLogger(syslogWrapper)
	mockPlugin.AssertCalled(t, "SetLogger", syslogWrapper)

	// Test GetFieldNames
	fieldNames := mockPlugin.GetFieldNames()
	assert.ElementsMatch(t, []string{"field1", "field2"}, fieldNames)

	// Test GetValues
	values := mockPlugin.GetValues("record")
	assert.ElementsMatch(t, []interface{}{1, "value"}, values)

	// Test Name
	name := mockPlugin.Name()
	assert.Equal(t, "TestPlugin", name)
}
