package main

import (
	"db-doc/database"
	"db-doc/model"
	"flag"
	"fmt"
	"os"
)

const version = "v1.1.1"

var dbConfig model.DbConfig

func main() {
	fmt.Printf("Welcome to the database document generation tool, the current version is %s \n", version)

	// 定义命令行参数
	dbTypePtr := flag.Int("db-type", 1, "Database type: 1:MySQL or MariaDB, 2:SQL Server, 3:PostgreSQL")
	hostPtr := flag.String("db-host", "127.0.0.1", "Database host")
	portPtr := flag.Int("db-port", 3306, "Database port")
	userPtr := flag.String("db-user", "root", "Database username")
	passwordPtr := flag.String("db-password", "123456", "Database password")
	databasePtr := flag.String("db-name", "", "Database name")
	docTypePtr := flag.Int("doc-type", 1, "Document type: 1:Online, 2:Offline")

	// 解析命令行参数
	flag.Parse()

	// 赋值给dbConfig
	dbConfig.DbType = *dbTypePtr
	dbConfig.Host = *hostPtr
	dbConfig.Port = *portPtr
	dbConfig.User = *userPtr
	dbConfig.Password = *passwordPtr
	dbConfig.Database = *databasePtr
	dbConfig.DocType = *docTypePtr

	// 检查DbType是否在有效范围内
	if dbConfig.DbType < 1 || dbConfig.DbType > 3 {
		fmt.Println("wrong number for --db-type, will exit ...")
		os.Exit(1)
	}

	// 根据DbType设置默认端口和用户名
	GetDefaultConfig()

	// generate
	database.Generate(&dbConfig)
}

// GetDefaultConfig get default config
func GetDefaultConfig() {
	// 根据DbType设置默认端口和用户名
	switch dbConfig.DbType {
	case 1:
		dbConfig.Port = 3306
		dbConfig.User = "root"
	case 2:
		dbConfig.Port = 1433
		dbConfig.User = "sa"
	case 3:
		dbConfig.Port = 5432
		dbConfig.User = "postgres"
	}
}
