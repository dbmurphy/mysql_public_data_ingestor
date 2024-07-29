package database

import (
	"database/sql"
	"fmt"
	"log"
	"mysql_public_data_ingestor/api_plugins"
	"mysql_public_data_ingestor/config"
	"mysql_public_data_ingestor/syslogwrapper"
)

type DBManager struct {
	DSN    string
	DBs    []string
	Tables map[string][]string
	dbConn *sql.DB
}

// NewDBManager initializes DBManager with the provided MySQL configuration and an optional dbConn.
// If dbConn is nil, it will create a new connection using DSN.
func NewDBManager(mysqlConfig config.MySQLConfig, dbConn ...*sql.DB) *DBManager {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?tls=%s",
		mysqlConfig.User, mysqlConfig.Password,
		mysqlConfig.Host, mysqlConfig.Port,
		mysqlConfig.DBName, mysqlConfig.TLS,
	)

	var conn *sql.DB
	if len(dbConn) > 0 && dbConn[0] != nil {
		conn = dbConn[0]
	} else {
		var err error
		conn, err = sql.Open("mysql", dsn)
		if err != nil {
			log.Fatalf("Failed to connect to MySQL: %v", err)
		}
	}

	return &DBManager{
		DSN:    dsn,
		dbConn: conn,
	}
}

func (dbm *DBManager) InitializeDatabases(cfg config.MainConfig, sysLog syslogwrapper.SyslogWrapperInterface, apiPlugin api_plugins.APIPlugin) {
	db := dbm.dbConn

	dbm.Tables = make(map[string][]string)

	for i := 1; i <= cfg.Databases.Copies; i++ {
		dbName := fmt.Sprintf("%s%d", cfg.Databases.Prefix, i)
		dbm.DBs = append(dbm.DBs, dbName)
		dbm.createDatabase(db, dbName, sysLog)
		tableName := apiPlugin.TablePrefix()
		dbm.createTable(db, dbName, tableName, sysLog, apiPlugin.Schema())
		dbm.Tables[dbName] = append(dbm.Tables[dbName], tableName)
	}

	for extraDB, dbConfig := range cfg.Databases.Extra {
		dbName := fmt.Sprintf("%s_%s", cfg.Databases.Prefix, extraDB)
		dbm.DBs = append(dbm.DBs, dbName)
		dbm.createDatabase(db, dbName, sysLog)
		for j := 1; j <= dbConfig.Tables; j++ {
			tableName := fmt.Sprintf("%s_%d", apiPlugin.TablePrefix(), j)
			dbm.createTable(db, dbName, tableName, sysLog, apiPlugin.Schema())
			dbm.Tables[dbName] = append(dbm.Tables[dbName], tableName)
		}
	}
}

func (dbm *DBManager) createDatabase(db *sql.DB, dbName string, sysLog syslogwrapper.SyslogWrapperInterface) {
	_, err := db.Exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s", dbName))
	if err != nil {
		sysLog.Warning(fmt.Sprintf("Failed to create database %s: %v", dbName, err))
	}
}

func (dbm *DBManager) createTable(db *sql.DB, dbName, tableName string, sysLog syslogwrapper.SyslogWrapperInterface, schema string) {
	_, err := db.Exec(fmt.Sprintf("USE %s", dbName))
	if err != nil {
		sysLog.Warning(fmt.Sprintf("Failed to use database %s: %v", dbName, err))
		return
	}

	createTableQuery := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s %s", tableName, schema)
	_, err = db.Exec(createTableQuery)
	if err != nil {
		sysLog.Warning(fmt.Sprintf("Failed to create table %s in database %s: %v", tableName, dbName, err))
	}
}
