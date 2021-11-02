package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"runtime/pprof"
	"sync"
	"time"

	"github.com/valyala/fasthttp"
)

type timeResponse struct {
	CurrentDateTime string `json:"currentDateTime"`
}

var memprofile = flag.String("memprofile", "", "write memory profile to `file`")
var reqs = flag.Int("reqs", 100, "write memory profile to `file`")

func main() {
	flag.Parse()

	var reqCount = *reqs

	var startTime = time.Now()
	var idCh = make(chan string, 100)
	var outCh = make(chan string, 100)
	var errCh = make(chan int, 1000)
	var workerWg = &sync.WaitGroup{}
	var wg = &sync.WaitGroup{}

	go func() {
		for ; reqCount > 0; reqCount-- {
			idCh <- ""
		}

		close(idCh)
	}()

	ctx, cancel := context.WithCancel(context.Background())

	wg.Add(1)
	go fillTime(ctx, outCh, wg)

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

	var errCount = 0
	for range errCh {
		errCount++
	}

	cancel()
	wg.Wait()

	fmt.Println("execution time: ", time.Since(startTime))
	fmt.Println("error count: ", errCount)

	memPeakUsage()

	if *memprofile != "" {
		f, err := os.Create(*memprofile)
		if err != nil {
			log.Fatal("could not create memory profile: ", err)
		}
		defer f.Close()

		runtime.GC()

		if err := pprof.WriteHeapProfile(f); err != nil {
			log.Fatal("could not write memory profile: ", err)
		}
	}
}

func handleWorker(idCh, outCh chan string, errCh chan int, wg *sync.WaitGroup) {
	defer wg.Done()

	client := &fasthttp.Client{
		MaxConnDuration:     10 * time.Second,
		MaxIdleConnDuration: 10 * time.Second,
		WriteTimeout:        10 * time.Second,
		ReadTimeout:         10 * time.Second,
		MaxConnWaitTimeout:  10 * time.Second,
		MaxConnsPerHost:     100000,
	}

	var tm = timeResponse{}
	var t time.Time
	for range idCh {
		req := fasthttp.AcquireRequest()
		resp := fasthttp.AcquireResponse()

		req.SetRequestURI("http://sam.wake-app.net/time")
		req.Header.Add("Connection", "Keep-Alive")

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
	}
}

func fillTime(ctx context.Context, outCh chan string, wg *sync.WaitGroup) {
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
		case <-ctx.Done():
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
