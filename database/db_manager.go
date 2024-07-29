package database

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/go-sql-driver/mysql"
	"mysql_public_data_ingestor/api_plugins"
	"mysql_public_data_ingestor/config"
	"mysql_public_data_ingestor/syslogwrapper"

	_ "github.com/go-sql-driver/mysql"
)

type DBManagerInterface interface {
	Conn(ctx context.Context) (*sql.Conn, error)
}

type DBManager struct {
	DSN    string
	DBs    []string
	Tables map[string][]string
	DbPool *sql.DB
}

func (dbm *DBManager) Conn(ctx context.Context) (*sql.Conn, error) {
	return dbm.DbPool.Conn(ctx)
}

// NewDBManager initializes DBManager with the provided MySQL configuration and creates a new connection pool.
func NewDBManager(mysqlConfig config.MySQLConfig) *DBManager {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?tls=%s",
		mysqlConfig.User, mysqlConfig.Password,
		mysqlConfig.Host, mysqlConfig.Port,
		mysqlConfig.DBName, setupTLSConfig(mysqlConfig.TLSConfig),
	)

	conn, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("Failed to connect to MySQL: %v", err)
	}

	// Apply connection pool settings directly
	conn.SetMaxOpenConns(mysqlConfig.ConnectionPool.MaxOpenConns)
	conn.SetMaxIdleConns(mysqlConfig.ConnectionPool.MaxIdleConns)
	conn.SetConnMaxLifetime(time.Duration(mysqlConfig.ConnectionPool.ConnMaxLifetime) * time.Second)

	return &DBManager{
		DSN:    dsn,
		DbPool: conn,
	}
}

func setupTLSConfig(tlsConfig config.TLSConfig) string {
	if tlsConfig.CAFile == "" && tlsConfig.CertFile == "" && tlsConfig.KeyFile == "" {
		return "false"
	}

	rootCertPool := x509.NewCertPool()
	if tlsConfig.CAFile != "" {
		pem, err := os.ReadFile(tlsConfig.CAFile)
		if err != nil {
			log.Fatalf("Failed to read CA file: %v", err)
		}
		if ok := rootCertPool.AppendCertsFromPEM(pem); !ok {
			log.Fatalf("Failed to append CA certificates")
		}
	}

	clientCert := make([]tls.Certificate, 0, 1)
	if tlsConfig.CertFile != "" && tlsConfig.KeyFile != "" {
		certs, err := tls.LoadX509KeyPair(tlsConfig.CertFile, tlsConfig.KeyFile)
		if err != nil {
			log.Fatalf("Failed to load client certificate and key: %v", err)
		}
		clientCert = append(clientCert, certs)
	}

	tlsConfigStruct := &tls.Config{
		RootCAs:            rootCertPool,
		Certificates:       clientCert,
		InsecureSkipVerify: tlsConfig.InsecureSkipVerify,
		ServerName:         tlsConfig.ServerName,
	}

	if tlsConfig.MinVersion != 0 {
		tlsConfigStruct.MinVersion = tlsConfig.MinVersion
	}
	if tlsConfig.MaxVersion != 0 {
		tlsConfigStruct.MaxVersion = tlsConfig.MaxVersion
	}
	if tlsConfig.CipherSuites != nil && len(tlsConfig.CipherSuites) > 0 {
		tlsConfigStruct.CipherSuites = tlsConfig.CipherSuites
	} else {
		tlsConfigStruct.CipherSuites = nil // Use default cipher suites
	}
	if tlsConfig.ClientAuth != 0 {
		tlsConfigStruct.ClientAuth = tls.ClientAuthType(tlsConfig.ClientAuth)
	}

	err := mysql.RegisterTLSConfig("custom", tlsConfigStruct)
	if err != nil {
		log.Fatalf("Failed to register custom TLS configuration: %v", err)
	}

	return "custom"
}

func (dbm *DBManager) InitializeDatabases(cfg config.MainConfig, sysLog syslogwrapper.SyslogWrapperInterface, apiPlugin api_plugins.APIPlugin) {
	db := dbm.DbPool

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
	query := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s`", dbName)
	_, err := db.Exec(query)
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

// PingIdleConnections pings all idle connections in the pool to keep them healthy
func (dbm *DBManager) PingIdleConnections(sysLog syslogwrapper.SyslogWrapperInterface) {
	for {
		time.Sleep(1 * time.Minute)
		idleConns := dbm.DbPool.Stats().Idle
		for i := 0; i < idleConns; i++ {
			conn, err := dbm.DbPool.Conn(context.Background())
			if err != nil {
				sysLog.Warning(fmt.Sprintf("Failed to get connection from pool: %v", err))
				continue
			}
			if err := conn.PingContext(context.Background()); err != nil {
				sysLog.Warning(fmt.Sprintf("Failed to ping database: %v", err))
			}
			err = conn.Close()
			if err != nil {
				sysLog.Warning(fmt.Sprintf("Failed to release DBPool database connection: %v", err))
			} // Release the connection back to the pool
		}
	}
}
