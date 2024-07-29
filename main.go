package mysql_public_data_ingestor

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"mysql_public_data_ingestor/api_plugins"
	"mysql_public_data_ingestor/config"
	"mysql_public_data_ingestor/database"
	"mysql_public_data_ingestor/syslogwrapper"
)

func main() {
	sysLog, err := SetupSyslog("data_pull")
	if err != nil {
		log.Fatalf("Failed to initialize syslog: %v", err)
	}
	defer sysLog.Close()

	cfg, err := LoadConfig(sysLog)
	if err != nil {
		log.Fatalf("Failed to load config file: %v", err)
	}

	apiPlugin, err := SetupPlugins(cfg, sysLog)
	if err != nil {
		log.Fatalf("Failed to setup plugins: %v", err)
	}

	dbManager, err := InitializeDatabases(cfg, sysLog, apiPlugin)
	if err != nil {
		log.Fatalf("Failed to initialize databases: %v", err)
	}

	tableChannels, wg := CreateTableWorkers(dbManager, sysLog, apiPlugin)

	stop := make(chan struct{})
	go StartDataFetching(apiPlugin, tableChannels, sysLog, stop)

	time.Sleep(1 * time.Minute) // Example: run for 1 minute
	close(stop)

	wg.Wait()
}

func SetupSyslog(tag string) (*syslogwrapper.SyslogWrapper, error) {
	return syslogwrapper.NewSyslogWrapper(tag)
}

func LoadConfig(sysLog syslogwrapper.SyslogWrapperInterface) (config.MainConfig, error) {
	configPath := os.Getenv("TEST_CONFIG_FILE")
	if configPath == "" {
		configPath = "config.yaml" // Default config file path
	}
	return config.LoadConfig(configPath, sysLog)
}

func SetupPlugins(cfg config.MainConfig, sysLog syslogwrapper.SyslogWrapperInterface) (api_plugins.APIPlugin, error) {
	err := api_plugins.LoadPlugins("api_plugins")
	if err != nil {
		sysLog.Error(fmt.Sprintf("Failed to load plugins: %v", err))
		return nil, err
	}

	api_plugins.SetLoggerForAllPlugins(sysLog)

	return api_plugins.InitPlugin(cfg.PluginSpec.Name)
}

func InitializeDatabases(cfg config.MainConfig, sysLog syslogwrapper.SyslogWrapperInterface, apiPlugin api_plugins.APIPlugin) (*database.DBManager, error) {
	dbManager := database.NewDBManager(cfg.MySQL)
	dbManager.InitializeDatabases(cfg, sysLog, apiPlugin)
	return dbManager, nil
}

func CreateTableWorkers(dbManager *database.DBManager, sysLog syslogwrapper.SyslogWrapperInterface, apiPlugin api_plugins.APIPlugin) (map[string]chan []interface{}, *sync.WaitGroup) {
	tableChannels := make(map[string]chan []interface{})
	var wg sync.WaitGroup

	for _, dbName := range dbManager.DBs {
		for _, tableName := range dbManager.Tables[dbName] {
			ch := make(chan []interface{})
			tableChannels[fmt.Sprintf("%s.%s", dbName, tableName)] = ch
			wg.Add(1)
			go TableWorker(dbName, tableName, ch, &wg, sysLog, dbManager.DSN, apiPlugin)
		}
	}

	return tableChannels, &wg
}

func StartDataFetching(apiPlugin api_plugins.APIPlugin, tableChannels map[string]chan []interface{}, sysLog syslogwrapper.SyslogWrapperInterface, stop chan struct{}) {
	go func() {
		for {
			select {
			case <-stop:
				for _, ch := range tableChannels {
					close(ch)
				}
				return
			default:
				err := FetchAndDistributeData(apiPlugin, tableChannels, sysLog)
				if err != nil {
					sysLog.Warning(fmt.Sprintf("Error fetching data: %v", err))
					time.Sleep(5 * time.Second) // Wait before retrying
					continue
				}
				interval, err := apiPlugin.Interval()
				if err != nil {
					sysLog.Warning(fmt.Sprintf("Error getting interval: %v", err))
					time.Sleep(5 * time.Second) // Wait before retrying
					continue
				}
				time.Sleep(time.Duration(interval) * time.Second)
			}
		}
	}()
}

func FetchAndDistributeData(apiPlugin api_plugins.APIPlugin, tableChannels map[string]chan []interface{}, sysLog syslogwrapper.SyslogWrapperInterface) error {
	data, err := apiPlugin.FetchData()
	if err != nil {
		return err
	}

	batchData := make([]interface{}, 0)
	switch d := data.(type) {
	case api_plugins.Response:
		for _, record := range d.Records {
			batchData = append(batchData, record)
		}
	default:
		return fmt.Errorf("unsupported data type")
	}

	for _, ch := range tableChannels {
		ch <- batchData
	}

	return nil
}

func TableWorker(dbName, tableName string, batchChan <-chan []interface{}, wg *sync.WaitGroup, sysLog syslogwrapper.SyslogWrapperInterface, dsn string, apiPlugin api_plugins.APIPlugin) {
	defer wg.Done()

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		sysLog.Warning(fmt.Sprintf("Failed to connect to MySQL: %v", err))
		return
	}
	defer func() {
		if err := db.Close(); err != nil {
			sysLog.Warning(fmt.Sprintf("Failed to close MySQL connection: %v", err))
		}
	}()

	fieldNames := apiPlugin.GetFieldNames()
	fieldNamesStr := strings.Join(fieldNames, ", ")
	placeholderStr := strings.Repeat("?, ", len(fieldNames)-1) + "?"

	for batch := range batchChan {
		for _, record := range batch {
			values := apiPlugin.GetValues(record)
			query := fmt.Sprintf("INSERT INTO %s.%s (%s) VALUES (%s)", dbName, tableName, fieldNamesStr, placeholderStr)
			_, err := db.Exec(query, values...)
			if err != nil {
				sysLog.Warning(fmt.Sprintf("Failed to insert record into %s.%s: %v", dbName, tableName, err))
				continue
			}
		}
	}
}