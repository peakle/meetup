package main

import (
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql" // add mysql driver

	sg "github.com/wakeapp/go-sql-generator"
)

type config struct {
	Host     string
	Username string
	Pass     string
	Port     string
	DBName   string
}

type SQLManager struct {
	conn *sql.DB
}

var m *SQLManager

// InitManager - init manager based on env params
func InitManager() (*SQLManager, error) {
	var err error

	if m == nil {
		m = &SQLManager{}

		err = m.open(&config{
			Host:     "db",
			Username: "deployer",
			Pass:     "deployer",
			Port:     "3306",
			DBName:   "meetup_db",
		})
		if err != nil {
			err = fmt.Errorf("on InitManager: %s", err.Error())
		}
	}

	return m, err
}

// InitCustomManager - init manager based on provided args
func InitCustomManager(host, username, password, DBname, port string) (*SQLManager, error) {
	manager := &SQLManager{}

	err := manager.open(&config{
		Host:     host,
		Username: username,
		Pass:     password,
		Port:     port,
		DBName:   DBname,
	})
	if err != nil {
		err = fmt.Errorf("on InitCustomManager: %s", err.Error())
	}

	return manager, err
}

// CloseManager - close connection to DB
func CloseManager() {
	_ = m.conn.Close()

	m = nil
}

// Insert - do insert
func (m *SQLManager) Insert(dataInsert *sg.InsertData) (int, error) {
	if len(dataInsert.ValuesList) == 0 {
		return 0, nil
	}

	sqlGenerator := sg.MysqlSqlGenerator{}

	query, args, err := sqlGenerator.GetInsertSql(*dataInsert)
	if err != nil {
		return 0, fmt.Errorf("on insert.generate insert sql: %s", err.Error())
	}

	var stmt *sql.Stmt
	stmt, err = m.conn.Prepare(query)
	if err != nil {
		return 0, fmt.Errorf("on insert.prepare stmt: %s", err.Error())
	}
	defer func() {
		_ = stmt.Close()
	}()

	var result sql.Result
	result, err = stmt.Exec(args...)
	if err != nil {
		return 0, fmt.Errorf("on insert.execute stmt: %s", err.Error())
	}

	ra, _ := result.RowsAffected()

	return int(ra), nil
}

// Upsert - do upsert
func (m *SQLManager) Upsert(dataUpsert *sg.UpsertData) (int, error) {
	if len(dataUpsert.ValuesList) == 0 {
		return 0, nil
	}

	sqlGenerator := sg.MysqlSqlGenerator{}

	query, args, err := sqlGenerator.GetUpsertSql(*dataUpsert)
	if err != nil {
		return 0, fmt.Errorf("on upsert.Generate query: %v, %s", dataUpsert, err.Error())
	}

	var stmt *sql.Stmt
	stmt, err = m.conn.Prepare(query)
	if err != nil {
		return 0, fmt.Errorf("on upsert.Prepare query: %s, %s", query, err.Error())
	}
	defer func() {
		_ = stmt.Close()
	}()

	var result sql.Result
	result, err = stmt.Exec(args...)
	if err != nil {
		return 0, fmt.Errorf("on upsert.Exec query, args: %v, %s", args, err.Error())
	}

	ra, _ := result.RowsAffected()

	return int(ra), nil
}

func (m *SQLManager) open(c *config) error {
	var conn *sql.DB
	var err error

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?collation=utf8_unicode_ci", c.Username, c.Pass, c.Host, c.Port, c.DBName)
	if conn, err = sql.Open("mysql", dsn); err != nil {
		return fmt.Errorf("on open connection to db: %s", err.Error())
	}

	m.conn = conn

	return nil
}

func (m *SQLManager) Query(sql string, args ...interface{}) (*sql.Rows, error) {
	return m.conn.Query(sql, args...)
}
