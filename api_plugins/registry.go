package api_plugins

import (
	"fmt"
	"mysql_public_data_ingestor/syslogwrapper"
	"path/filepath"
	"plugin"
)

var registry = make(map[string]APIPlugin)

func Register(name string, apiPlugin APIPlugin) {
	registry[name] = apiPlugin
}

func InitPlugin(name string) (APIPlugin, error) {
	apiPlugin, exists := registry[name]
	if !exists {
		return nil, fmt.Errorf("unsupported API plugin: %s", name)
	}
	return apiPlugin, nil
}

func LoadPlugins(dir string) error {
	files, err := filepath.Glob(filepath.Join(dir, "*.so"))
	if err != nil {
		return err
	}

	for _, file := range files {
		plg, err := plugin.Open(file)
		if err != nil {
			return fmt.Errorf("error loading plugin %s: %v", file, err)
		}

		sym, err := plg.Lookup("PluginInstance")
		if err != nil {
			return fmt.Errorf("error loading symbol from plugin %s: %v", file, err)
		}

		apiPlugin, ok := sym.(APIPlugin)
		if !ok {
			return fmt.Errorf("plugin %s does not implement APIPlugin interface", file)
		}

		Register(apiPlugin.Name(), apiPlugin)
	}
	return nil
}

func SetLoggerForAllPlugins(sysLog syslogwrapper.SyslogWrapperInterface) {
	for _, apiPlugin := range registry {
		apiPlugin.SetLogger(sysLog)
	}
}
