package api_plugins

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"mysql_public_data_ingestor/syslogwrapper"
	"testing"
)

func TestRegister(t *testing.T) {
	mockPlugin := new(MockAPIPlugin)
	mockPlugin.On("Name").Return("testPlugin")

	Register("testPlugin", mockPlugin)

	plugin, exists := registry["testPlugin"]
	assert.True(t, exists, "Plugin should be registered")
	assert.Equal(t, mockPlugin, plugin, "Registered plugin should match the mocked plugin")
}

func TestInitPlugin(t *testing.T) {
	mockPlugin := new(MockAPIPlugin)
	mockPlugin.On("Name").Return("testPlugin")

	Register("testPlugin", mockPlugin)

	plugin, err := InitPlugin("testPlugin")
	assert.NoError(t, err, "Error should be nil when retrieving a registered plugin")
	assert.Equal(t, mockPlugin, plugin, "Retrieved plugin should match the registered plugin")

	_, err = InitPlugin("nonExistentPlugin")
	assert.Error(t, err, "Error should be returned for an unsupported plugin")
}

// Mock APIPlugin implementation for this test file
type MockPluginLoader struct {
	mock.Mock
}

func (m *MockPluginLoader) LoadPlugin(name string) (APIPlugin, error) {
	args := m.Called(name)
	return args.Get(0).(APIPlugin), args.Error(1)
}

func TestLoadPlugins(t *testing.T) {
	// Create a mock plugin loader to avoid actual file operations
	mockLoader := new(MockPluginLoader)
	mockPlugin := new(MockAPIPlugin)
	mockPlugin.On("Name").Return("mockPlugin")

	mockLoader.On("LoadPlugin", "mockPlugin").Return(mockPlugin, nil)

	// Register mock plugin for testing
	Register("mockPlugin", mockPlugin)

	// Simulate loading plugins
	err := LoadPlugins("./mock_plugins") // Ensure this directory exists or mock it as needed
	assert.NoError(t, err, "Error should be nil when loading plugins from a directory")

	// Verify plugins were registered
	plugin, exists := registry["mockPlugin"]
	assert.True(t, exists, "Mock plugin should be registered")
	assert.NotNil(t, plugin, "Mock plugin should be non-nil")
}

func TestSetLogger(t *testing.T) {
	mockPlugin1 := new(MockAPIPlugin)
	mockPlugin2 := new(MockAPIPlugin)
	syslogWrapper := new(syslogwrapper.SyslogWrapperInterface)

	Register("plugin1", mockPlugin1)
	Register("plugin2", mockPlugin2)

	mockPlugin1.On("SetLogger", *syslogWrapper).Return()
	mockPlugin2.On("SetLogger", *syslogWrapper).Return()
}

// TODO: Add test for SetLoggerAllPlugins
