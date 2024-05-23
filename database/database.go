package database

import (
	"database/sql"
	"db-doc/doc"
	"db-doc/model"
	"fmt"
	"os"

	_ "github.com/denisenkom/go-mssqldb"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

var dbConfig model.DbConfig

// Generate generate doc
func Generate(config *model.DbConfig) {
	dbConfig = *config
	db := initDB()
	if db == nil {
		fmt.Println("init database err")
		os.Exit(1)
	}
	defer db.Close()
	var dbInfo model.DbInfo
	dbInfo.DbName = config.Database
	tables := getTableInfo(db)
	// create
	doc.CreateDoc(dbInfo, config.DocType, tables)
}

// InitDB 初始化数据库
func initDB() *sql.DB {
	var (
		dbURL  string
		dbType string
	)
	if dbConfig.DbType == 1 {
		// https://github.com/go-sql-driver/mysql/
		dbType = "mysql"
		// <username>:<password>@<host>:<port>/<database>
		dbURL = fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8&parseTime=True&loc=Local",
			dbConfig.User, dbConfig.Password, dbConfig.Host, dbConfig.Port, dbConfig.Database)
	}
	if dbConfig.DbType == 2 {
		// https://github.com/denisenkom/go-mssqldb
		dbType = "mssql"
		// server=%s;database=%s;user id=%s;password=%s;port=%d;encrypt=disable
		dbURL = fmt.Sprintf("server=%s;database=%s;user id=%s;password=%s;port=%d;encrypt=disable",
			dbConfig.Host, dbConfig.Database, dbConfig.User, dbConfig.Password, dbConfig.Port)
	}
	if dbConfig.DbType == 3 {
		// https://github.com/lib/pq
		dbType = "postgres"
		// postgres://pqgotest:password@localhost:5432/pqgotest?sslmode=verify-full
		dbURL = fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable", dbConfig.User, dbConfig.Password,
			dbConfig.Host, dbConfig.Port, dbConfig.Database)
	}
	db, err := sql.Open(dbType, dbURL)
	if err != nil {
		fmt.Println(err)
	}
	err = db.Ping()
	if err != nil {
		panic(err.Error()) // proper error handling instead of panic in your app
	}
	return db
}

// getDbInfo 获取数据库的基本信息
func getDbInfo(db *sql.DB) model.DbInfo {
	var (
		info       model.DbInfo
		rows       *sql.Rows
		err        error
		key, value string
	)
	// 数据库版本
	rows, err = db.Query("SELECT @@version;")
	if err != nil {
		fmt.Println(err)
	}
	defer rows.Close()
	for rows.Next() {
		rows.Scan(&value)
	}
	info.Version = value
	// 字符集
	rows, err = db.Query("SHOW variables LIKE '%character_set_server%';")
	if err != nil {
		fmt.Println(err)
	}
	defer rows.Close()
	for rows.Next() {
		rows.Scan(&key, &value)
	}
	info.Charset = value
	// 排序规则
	rows, err = db.Query("SHOW variables LIKE 'collation_server%';")
	if err != nil {
		fmt.Println(err)
	}
	defer rows.Close()
	for rows.Next() {
		rows.Scan(&key, &value)
	}
	info.Collation = value
	return info
}

// getTableInfo 获取表信息
func getTableInfo(db *sql.DB) []model.Table {
	// find all tables
	tables := make([]model.Table, 0)
	rows, err := db.Query(getTableSQL())
	if err != nil {
		fmt.Println(err)
	}
	defer rows.Close()
	var table model.Table
	for rows.Next() {
		table.TableComment = ""
		rows.Scan(&table.TableName, &table.TableComment)
		if len(table.TableComment) == 0 {
			table.TableComment = table.TableName
		}
		tables = append(tables, table)
	}
	for i := range tables {
		columns := getColumnInfo(db, tables[i].TableName)
		tables[i].ColList = columns
	}
	return tables
}

// getColumnInfo 获取列信息
func getColumnInfo(db *sql.DB, tableName string) []model.Column {
	columns := make([]model.Column, 0)
	rows, err := db.Query(getColumnSQL(tableName))
	if err != nil {
		fmt.Println(err)
	}
	var column model.Column
	for rows.Next() {
		rows.Scan(&column.ColName, &column.ColType, &column.ColKey, &column.IsNullable, &column.ColComment, &column.ColDefault)
		columns = append(columns, column)
		column.ColDefault = ""
	}
	return columns
}

// getTableSQL
func getTableSQL() string {
	var sql string
	if dbConfig.DbType == 1 {
		sql = fmt.Sprintf(`
			SELECT table_name    AS TableName, 
			       table_comment AS TableComment
			FROM information_schema.tables 
			WHERE table_schema = '%s'
		`, dbConfig.Database)
	}
	if dbConfig.DbType == 2 {
		sql = fmt.Sprintf(`
		SELECT * FROM (
			SELECT CAST(so.name AS varchar(500)) AS TableName, 
			CAST(sep.value AS varchar(500))      AS TableComment
			FROM sysobjects so
			LEFT JOIN sys.extended_properties sep ON sep.major_id=so.id AND sep.minor_id=0
			WHERE (xtype='U' OR xtype='v')
		) t 
		`)
	}
	if dbConfig.DbType == 3 {
		sql = fmt.Sprintf(`
			SELECT a.relname     AS TableName, 
				   b.description AS TableComment
			FROM pg_class a
			LEFT OUTER JOIN pg_description b ON b.objsubid = 0 AND a.oid = b.objoid
			WHERE a.relnamespace = (SELECT oid FROM pg_namespace WHERE nspname = 'public')
			AND a.relkind = 'r'
			ORDER BY a.relname
		`)
	}
	return sql
}

// getColumnSQL
func getColumnSQL(tableName string) string {
	var sql string
	if dbConfig.DbType == 1 {
		sql = fmt.Sprintf(`
			SELECT column_name AS ColName,
			column_type        AS ColType,
			column_key         AS ColKey,
			is_nullable        AS IsNullable,
			column_comment     AS ColComment,
			column_default     AS ColDefault
			FROM information_schema.columns 
			WHERE table_schema = '%s' AND table_name = '%s' ORDER BY ordinal_position
		`, dbConfig.Database, tableName)
	}
	if dbConfig.DbType == 2 {
		sql = fmt.Sprintf(`
		SELECT 
			ColName = a.name,
			ColType = b.name + '(' + CAST(COLUMNPROPERTY(a.id, a.name, 'PRECISION') AS varchar) + ')',
			ColKey  = CASE WHEN EXISTS(SELECT 1
										FROM sysobjects
										WHERE xtype = 'PK'
										AND name IN (
											SELECT name
											FROM sysindexes
											WHERE indid IN (
												SELECT indid
												FROM sysindexkeys
												WHERE id = a.id AND colid = a.colid
										))) THEN 'PRI'
							ELSE '' END,
			IsNullable = CASE WHEN a.isnullable = 1 THEN 'YES' ELSE 'NO' END,
			ColComment = ISNULL(g.[VALUE], ''),
			ColDefault = ISNULL(e.text, '')
		FROM syscolumns a
				LEFT JOIN systypes b ON a.xusertype = b.xusertype
				INNER JOIN sysobjects d ON a.id = d.id AND d.xtype = 'U' AND d.name <> 'dtproperties'
				LEFT JOIN syscomments e ON a.cdefault = e.id
				LEFT JOIN sys.extended_properties g ON a.id = g.major_id AND a.colid = g.minor_id
				LEFT JOIN sys.extended_properties f ON d.id = f.major_id AND f.minor_id = 0
		WHERE d.name = '%s'
		ORDER BY a.id, a.colorder
		`, tableName)
	}
	if dbConfig.DbType == 3 {
		sql = fmt.Sprintf(`
		SELECT
			column_name AS ColName,
			data_type AS ColType,
			CASE
				WHEN b.pk_name IS NULL THEN ''
				ELSE 'PRI'
			END AS ColKey,
			is_nullable AS IsNullable,
			c.DeText AS ColComment,
			column_default AS ColDefault
		FROM
			information_schema.columns
		LEFT JOIN (
			SELECT
				pg_attr.attname AS colname,
				pg_constraint.conname AS pk_name
			FROM
				pg_constraint
			INNER JOIN pg_class ON
				pg_constraint.conrelid = pg_class.oid
			INNER JOIN pg_attribute pg_attr ON
				pg_attr.attrelid = pg_class.oid
				AND pg_attr.attnum = pg_constraint.conkey[1]
			INNER JOIN pg_type ON
				pg_type.oid = pg_attr.atttypid
			WHERE
				pg_class.relname = 'file_sources'
				AND pg_constraint.contype = 'p' ) b ON
			b.colname = information_schema.columns.column_name
		LEFT JOIN (
			SELECT
				attname,
				description AS DeText
			FROM
				pg_class
			LEFT JOIN pg_attribute pg_attr ON
				pg_attr.attrelid = pg_class.oid
			LEFT JOIN pg_description pg_desc ON
				pg_desc.objoid = pg_attr.attrelid
				AND pg_desc.objsubid = pg_attr.attnum
			WHERE
				pg_attr.attnum>0
				AND pg_attr.attrelid = pg_class.oid
				AND pg_class.relname = 'file_sources' )c ON
			c.attname = information_schema.columns.column_name
		WHERE
			table_schema = 'public'
			AND table_name = '%s'
		ORDER BY
			ordinal_position DESC`, tableName)
	}
	return sql
}
