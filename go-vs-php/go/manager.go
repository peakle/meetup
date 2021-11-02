package main

import (
	"database/sql"
	"fmt"
	"strconv"

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

func insertTime(ros []string) error {
	var err error
	var m *SQLManager
	var c string

	m, err = InitManager()
	if err != nil {
		return fmt.Errorf("on InsertToDb.InitManager: %s", err.Error())
	}

	var d = &sg.InsertData{
		TableName: "Parsing",
		Fields: []string{
			"time",
			"t",
		},
		IsIgnore: true,
	}

	var count int
	for _, c = range ros {
		d.Add([]string{
			c,
			strconv.Itoa(count),
		})
		count++
	}

	_, err = m.Insert(d)
	if err != nil {
		return fmt.Errorf("on InsertToDb: %s", err.Error())
	}

	return nil
}

// InitManager - init manager based on env params
func InitManager() (*SQLManager, error) {
	var err error

	if m == nil {
		m = &SQLManager{}

		err = m.open(&config{
			Host:     "127.0.0.1",
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
