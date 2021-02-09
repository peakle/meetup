package main

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/valyala/fasthttp"
	sg "github.com/wakeapp/go-sql-generator"
)

type timeResponse struct {
	CurrentDateTime string `json:"currentDateTime"`
}

func main() {
	// err := pprof.StartCPUProfile(os.Stdout)
	// if err != nil {
	// fmt.Println("on start cpuUsage: ", err.Error())
	// }

	scanner := bufio.NewScanner(os.Stdin)

	scanner.Scan()
	testName := scanner.Text() // test num

	if testName == "1" { // db bencmark
		scanner.Scan()
		var thread, _ = strconv.Atoi(scanner.Text()) // thread

		scanner.Scan()
		var threadCount, _ = strconv.Atoi(scanner.Text()) // threadCount

		var startTime = time.Now()
		var idCh = make(chan string, 100)
		var outCh = make(chan string, 100)
		var doneCh = make(chan struct{}, 1)
		var workerWg = &sync.WaitGroup{}
		var wg = &sync.WaitGroup{}

		var ids = queryNonGrouped(thread, threadCount)

		maxWorker := len(ids) / 5
		if maxWorker <= 0 {
			maxWorker = 1
		}

		for workerCount := 0; workerCount < maxWorker; workerCount++ {
			workerWg.Add(1)
			go handle(idCh, outCh, workerWg)
		}

		for id := range ids {
			idCh <- id
		}

		go fillGrouped(outCh, doneCh, wg)

		close(idCh)
		workerWg.Wait()

		doneCh <- struct{}{} // release fillCommissionUpsert
		wg.Wait()

		fmt.Println("execution time: ", time.Since(startTime))
	} else if testName == "2" { // parsing bencmark
		scanner.Scan()
		var reqCount, _ = strconv.Atoi(scanner.Text()) // reqCount

		var startTime = time.Now()

		var idCh = make(chan string, 100)
		var outCh = make(chan string, 100)
		var doneCh = make(chan struct{}, 1)
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
			go handleWorker(idCh, outCh, workerWg)
		}

		workerWg.Wait()
		close(outCh)

		doneCh <- struct{}{} // release fillTime

		wg.Wait()

		fmt.Println("execution time: ", time.Since(startTime))
	} else {
		fmt.Println("wrong bencmark name")
	}

	memPeakUsage()

	// pprof.StopCPUProfile()
}

func handleWorker(idCh, outCh chan string, wg *sync.WaitGroup) {
	defer wg.Done()

	client := &fasthttp.Client{}

	var tm = timeResponse{}
	var t time.Time
	for range idCh {
		req := fasthttp.AcquireRequest()
		resp := fasthttp.AcquireResponse()

		req.SetRequestURI("http://worldclockapi.com/api/json/est/now")
		req.SetConnectionClose()

		err := client.DoTimeout(req, resp, 10*time.Second)
		if err != nil {
			fmt.Println("on handleWorker.Do: ", err.Error())
			fasthttp.ReleaseRequest(req)
			fasthttp.ReleaseResponse(resp)
			continue
		}

		err = json.Unmarshal(resp.Body(), &tm)
		if err != nil {
			fmt.Println("on handleWorker.Unmarshal: ", err.Error())
			fasthttp.ReleaseRequest(req)
			fasthttp.ReleaseResponse(resp)
			continue
		}

		t, err = time.Parse("2006-02-01T15:04-07:00", tm.CurrentDateTime)
		if err != nil {
			fmt.Println("on handleWorker.time.Parse: ", err.Error())
			continue
		}

		outCh <- t.String()

		fasthttp.ReleaseRequest(req)
		fasthttp.ReleaseResponse(resp)

		time.Sleep(500 * time.Millisecond)
	}
}

func handle(idCh, outCh chan string, wg *sync.WaitGroup) {
	defer wg.Done()

	for id := range idCh {
		time.Sleep(time.Millisecond * 500)
		outCh <- id
	}
}

func fillTime(outCh chan string, doneCh chan struct{}, wg *sync.WaitGroup) {
	defer wg.Done()

	const maxUpsertLen = 1000

	var err error
	var timeList = make([]string, 0, maxUpsertLen)
	var tm string
	var ok bool

	_, err = InitManager()
	if err != nil {
		log.Fatalf("on fillTime: %s", err.Error())
	}
	defer CloseManager()

	var ticker = time.NewTicker(time.Second * 3)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if len(timeList) > 0 {
				err = insertTime(timeList)
				if err != nil {
					log.Printf("on InsertToDbTime: %s \n", err.Error())
					continue
				}

				timeList = timeList[:0]
			}

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

func fillGrouped(outCh chan string, doneCh chan struct{}, wg *sync.WaitGroup) {
	defer wg.Done()

	const maxUpsertLen = 5000

	var err error
	var ids = make([]string, 0, maxUpsertLen)
	var id string
	var ok bool

	_, err = InitManager()
	if err != nil {
		log.Fatalf("on fillTime: %s", err.Error())
	}
	defer CloseManager()

	for {
		select {
		case id, ok = <-outCh:
			if len(ids) >= maxUpsertLen {
				err = insertGrouped(ids)
				if err != nil {
					log.Printf("on insertGrouped: %s \n", err.Error())
					continue
				}

				ids = ids[:0]
			}

			if ok {
				ids = append(ids, id)
				continue
			}

			if id != "" {
				ids = append(ids, id)
			}

			for id = range outCh {
				ids = append(ids, id)
			}

			outCh = nil
		case <-doneCh:
			if outCh != nil {
				for id = range outCh {
					ids = append(ids, id)
				}
			}

			if len(ids) > 0 {
				err = insertGrouped(ids)
				if err != nil {
					log.Printf("on fillCommissionUpsert: %s \n", err.Error())
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
		},
		IsIgnore: true,
	}

	for _, c = range ros {
		d.Add([]string{
			c,
		})
	}

	_, err = m.Insert(d)
	if err != nil {
		return fmt.Errorf("on InsertToDb: %s", err.Error())
	}

	return nil
}

func insertGrouped(ros []string) error {
	var err error
	var m *SQLManager
	var c string

	m, err = InitManager()
	if err != nil {
		return fmt.Errorf("on InsertToDb.InitManager: %s", err.Error())
	}

	var d = &sg.InsertData{
		TableName: "Grouped",
		Fields: []string{
			"id",
		},
		IsIgnore: true,
	}

	for _, c = range ros {
		d.Add([]string{
			c,
		})
	}

	_, err = m.Insert(d)
	if err != nil {
		return fmt.Errorf("on InsertToDb: %s", err.Error())
	}
	return nil
}

func queryNonGrouped(thread, threadCount int) map[string]string {
	q := fmt.Sprintf("SELECT id FROM NonGrouped WHERE mod(id, %d) = %d", threadCount, thread-1)

	var _, err = InitManager()
	if err != nil {
		log.Fatal("on query: ", err)
	}

	rows, err := m.Query(q)
	if err != nil {
		if err == sql.ErrNoRows {
			return map[string]string{}
		}

		log.Fatalf("on query.Query: %s", err.Error())
	}

	var id string

	var res = make(map[string]string, 10)

	for rows.Next() {
		err = rows.Scan(&id)

		res[id] = id
		if err != nil {
			log.Fatalf("on on query.Scan: %s", err.Error())
		}
	}

	return res
}

func memPeakUsage() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	fmt.Printf("Alloc memory = %v MB \n", bToMb(m.TotalAlloc))
	fmt.Printf("Sys memory = %v MB \n", bToMb(m.Sys))
}

func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}
