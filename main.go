package main

import (
	"context"
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

	go dbManager.PingIdleConnections(sysLog) // Keep the connection pool healthy

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
			go TableWorker(dbName, tableName, ch, &wg, sysLog, dbManager, apiPlugin)
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
	// Fetch data from the API plugin
	data, err := apiPlugin.FetchData()
	if err != nil {
		return err
	}

	// Convert fetched data to batch data
	var batchData []interface{}
	switch d := data.(type) {
	case api_plugins.Response:
		for _, record := range d.Records {
			batchData = append(batchData, record)
		}
	default:
		sysLog.Warning(fmt.Sprintf("FetchAndDistributeData: Unsupported data type: %T", data))
		return fmt.Errorf("unsupported data type")
	}

	// Send the batch data to each channel
	for _, ch := range tableChannels {
		// Send the batch data to the channel
		// Here we use a goroutine to avoid blocking if the channel might be full
		go func(ch chan []interface{}) {
			ch <- batchData
		}(ch)
	}

	return nil
}

func TableWorker(dbName, tableName string, batchChan <-chan []interface{}, wg *sync.WaitGroup, sysLog syslogwrapper.SyslogWrapperInterface, dbManager database.DBManagerInterface, apiPlugin api_plugins.APIPlugin) {
	defer wg.Done()

	fieldNames := apiPlugin.GetFieldNames()
	fieldNamesStr := strings.Join(fieldNames, ", ")
	placeholderStr := strings.Repeat("?, ", len(fieldNames)-1) + "?"

	// Get a connection from the pool
	db, err := dbManager.Conn(context.Background())
	if err != nil {
		sysLog.Warning(fmt.Sprintf("Failed to get connection from pool: %v", err))
		return
	}
	defer func(db *sql.Conn) {
		err := db.Close()
		if err != nil {
			sysLog.Warning(fmt.Sprintf("Failed to release DBPool connection: %v", err))
		}
	}(db)

	for batch := range batchChan {
		tx, err := db.BeginTx(context.Background(), &sql.TxOptions{})
		if err != nil {
			sysLog.Warning(fmt.Sprintf("Failed to begin transaction: %v", err))
			continue
		}

		for _, record := range batch {
			values := apiPlugin.GetValues(record)
			query := fmt.Sprintf("%s %s.%s (%s) VALUES (%s)",
				"INSERT INTO",
				dbName,
				tableName,
				fieldNamesStr,
				placeholderStr,
			)
			_, err := tx.Exec(query, values...)
			if err != nil {
				sysLog.Warning(fmt.Sprintf("Failed to insert record into %s.%s: %v", dbName, tableName, err))
				err := tx.Rollback()
				if err != nil {
					sysLog.Warning(fmt.Sprintf("Failed to rollback transaction: %v", err))
				} // Rollback the current transaction on error
				continue
			}
			// Commit each record change
			err = tx.Commit()
			if err != nil {
				sysLog.Warning(fmt.Sprintf("Failed to commit transaction: %v", err))
			}
		}
	}
}
