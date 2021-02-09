package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"runtime/pprof"
	"strconv"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql" // add mysql driver

	"github.com/valyala/fasthttp"
	sg "github.com/wakeapp/go-sql-generator"
)

type timeResponse struct {
	CurrentDateTime string `json:"currentDateTime"`
}

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
var memprofile = flag.String("memprofile", "", "write memory profile to `file`")
var reqs = flag.Int("reqs", 100, "write memory profile to `file`")

func main() {
	flag.Parse()

	var reqCount = *reqs

	var startTime = time.Now()
	var idCh = make(chan string, 100)
	var outCh = make(chan string, 100)
	var doneCh = make(chan struct{}, 1)
	var errCh = make(chan int, 1000)
	var workerWg = &sync.WaitGroup{}
	var wg = &sync.WaitGroup{}

	go func() {
		for ; reqCount > 0; reqCount-- {
			idCh <- ""
		}

		close(idCh)
	}()

	wg.Add(1)
	go fillTime(outCh, doneCh, wg)

	var maxWorker = reqCount
	if maxWorker <= 0 {
		maxWorker = 1
	}

	if maxWorker > 400 {
		maxWorker = 400
	}

	for workerCount := 0; workerCount < maxWorker; workerCount++ {
		workerWg.Add(1)
		go handleWorker(idCh, outCh, errCh, workerWg)
	}

	workerWg.Wait()
	close(outCh)
	close(errCh)

	errCount := 0
	wg.Add(1)
	go func() {
		defer wg.Done()
		for range errCh {
			errCount++
		}
	}()

	doneCh <- struct{}{} // release fillTime

	wg.Wait()

	fmt.Println("execution time: ", time.Since(startTime))
	fmt.Println("errorCount time: ", errCount)

	memPeakUsage()

	if *memprofile != "" {
		f, err := os.Create(*memprofile)
		if err != nil {
			log.Fatal("could not create memory profile: ", err)
		}
		defer f.Close() // error handling omitted for example
		runtime.GC()    // get up-to-date statistics
		if err := pprof.WriteHeapProfile(f); err != nil {
			log.Fatal("could not write memory profile: ", err)
		}
	}
}

func handleWorker(idCh, outCh chan string, errCh chan int, wg *sync.WaitGroup) {
	defer wg.Done()

	client := &fasthttp.Client{}

	var tm = timeResponse{}
	var t time.Time
	for range idCh {
		req := fasthttp.AcquireRequest()
		resp := fasthttp.AcquireResponse()

		req.SetRequestURI("http://sam.wake-app.net/time")

		req.SetConnectionClose()
		resp.SetConnectionClose()

		err := client.DoTimeout(req, resp, 10*time.Second)
		if err != nil {
			fmt.Println("on handleWorker.Do: ", err.Error())
			fasthttp.ReleaseRequest(req)
			fasthttp.ReleaseResponse(resp)
			errCh <- 1
			continue
		}

		err = json.Unmarshal(resp.Body(), &tm)
		if err != nil {
			fmt.Println("on handleWorker.Unmarshal: ", err.Error())
			fasthttp.ReleaseRequest(req)
			fasthttp.ReleaseResponse(resp)
			errCh <- 1

			continue
		}

		t, err = time.Parse("2006-02-01T15:04-07:00", tm.CurrentDateTime)
		if err != nil {
			fmt.Println("on handleWorker.time.Parse: ", err.Error())
			errCh <- 1

			continue
		}

		outCh <- t.String()

		fasthttp.ReleaseRequest(req)
		fasthttp.ReleaseResponse(resp)

		time.Sleep(500 * time.Millisecond)
	}
}

func fillTime(outCh chan string, doneCh chan struct{}, wg *sync.WaitGroup) {
	defer wg.Done()

	const maxUpsertLen = 5000

	var err error
	var timeList = make([]string, 0, maxUpsertLen)
	var tm string
	var ok bool

	_, err = InitManager()
	if err != nil {
		log.Fatalf("on fillTime: %s", err.Error())
	}
	defer CloseManager()

	for {
		select {
		case tm, ok = <-outCh:
			if len(timeList) >= maxUpsertLen {
				err = insertTime(timeList)
				if err != nil {
					log.Printf("on InsertToDbTime: %s \n", err.Error())
					continue
				}

				timeList = timeList[:0]
			}

			if ok {
				timeList = append(timeList, tm)
				continue
			}

			if tm != "" {
				timeList = append(timeList, tm)
			}

			for tm = range outCh {
				timeList = append(timeList, tm)
			}

			outCh = nil
		case <-doneCh:
			if outCh != nil {
				for tm = range outCh {
					timeList = append(timeList, tm)
				}
			}

			if len(timeList) > 0 {
				err = insertTime(timeList)
				if err != nil {
					log.Printf("on InsertToDbTime: %s \n", err.Error())
				}
			}

			return
		}
	}
}

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

func memPeakUsage() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	fmt.Printf("Total Alloc memory = %v MB \n", bToMb(m.TotalAlloc))
	fmt.Printf("Alloc memory = %v MB \n", bToMb(m.Alloc))
	fmt.Printf("Sys memory = %v MB \n", bToMb(m.Sys))
}

func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
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
