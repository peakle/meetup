package go_optimizations

import (
	"database/sql"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type RowHeap struct {
	Str *string `db:"str"`
}

type RowStack struct {
	Str sql.NullString `db:"str"`
}

// global var disable compiler loop optimization
var (
	globalRowHeap    RowHeap
	globalPtrRowHeap *RowHeap
	globalRowStack   RowStack
)

func BenchmarkCreateRowHeap(b *testing.B) {
	for i := 0; i < b.N; i++ {
		globalRowHeap = CreateRowHeap()
	}
}
func BenchmarkCreateRowHeapOnHeap(b *testing.B) {
	ptrRowHeap := func(h RowHeap) *RowHeap { return &h }
	for i := 0; i < b.N; i++ {
		globalPtrRowHeap = ptrRowHeap(CreateRowHeap())
	}
}

func CreateRowHeap() RowHeap {
	k := "foo"
	return RowHeap{Str: &k}
}

func BenchmarkCreateRowStack(b *testing.B) {
	for i := 0; i < b.N; i++ {
		globalRowStack = CreateRowStack()
	}
}

func CreateRowStack() RowStack {
	k := "foo"
	return RowStack{Str: sql.NullString{
		String: k,
		Valid:  true,
	}}
}

// go test --bench=BenchmarkGoClosureInLoop --benchmem --count=3
func BenchmarkGoClosureInLoop(b *testing.B) {
	b.Run("go_closure_in_loop", func(b *testing.B) {
		var wg sync.WaitGroup
		for i := 0; i < b.N; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				dummyProcess(benchCount / 2)
			}()
			wg.Wait()
		}
	})

	b.Run("go_closure", func(b *testing.B) {
		var wg sync.WaitGroup
		worker := func() {
			defer wg.Done()
			dummyProcess(benchCount / 2)
		}

		for i := 0; i < b.N; i++ {
			wg.Add(1)
			go worker()
			wg.Wait()
		}
	})
}

func BenchmarkTryLock(b *testing.B) {
	const maxBackoff = 16
	var (
		wg   sync.WaitGroup
		mu   = &sync.Mutex{}
		lock int32
	)

	// only for > go 1.18.1
	//runnerParallel(b, "try_lock", 100000, func(i int64) {
	//	backoff := 1
	//
	//	for !mu.TryLock() {
	//		for i := 0; i < backoff; i++ {
	//			runtime.Gosched()
	//		}
	//		if backoff < maxBackoff {
	//			backoff <<= 1
	//		}
	//	}
	//	mu.Unlock()
	//}, &wg)

	runnerParallel(b, "mutex", 100000, func(i int64) {
		mu.Lock()
		mu.Unlock()
	}, &wg)

	runnerParallel(b, "cas", 100000, func(i int64) {
		backoff := 1

		for !atomic.CompareAndSwapInt32(&lock, 0, 1) {
			for i := 0; i < backoff; i++ {
				runtime.Gosched()
			}
			if backoff < maxBackoff {
				backoff <<= 1
			}
		}
		atomic.StoreInt32(&lock, 0)
	}, &wg)
}

func BenchmarkTicker(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = time.NewTicker(time.Second)
	}
	stats := checkMem()
	b.Logf("memory usage: %d MB", stats.TotalAlloc/MiB)
	b.Logf("GC cycles: %d", stats.NumGC)
	b.Logf("Heap size: %d", stats.HeapObjects)
	b.Logf("goroutines count: %d", runtime.NumGoroutine())
}

func BenchmarkTickerWithStop(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ticker := time.NewTicker(time.Second)
		ticker.Stop()
	}
	stats := checkMem()
	b.Logf("memory usage: %d MB", stats.TotalAlloc/MiB)
	b.Logf("GC cycles: %d", stats.NumGC)
	b.Logf("Heap size: %d", stats.HeapObjects)
	b.Logf("goroutines count: %d", runtime.NumGoroutine())
}

func BenchmarkTimer(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = time.NewTimer(time.Second)
	}
	stats := checkMem()
	b.Logf("memory usage: %d MB", stats.TotalAlloc/MiB)
	b.Logf("GC cycles: %d", stats.NumGC)
	b.Logf("Heap size: %d", stats.HeapObjects)
	b.Logf("goroutines count: %d", runtime.NumGoroutine())
}

func BenchmarkTimerWithStop(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ticker := time.NewTimer(time.Second)
		ticker.Stop()
	}
	stats := checkMem()
	b.Logf("memory usage: %d MB", stats.TotalAlloc/MiB)
	b.Logf("GC cycles: %d", stats.NumGC)
	b.Logf("Heap size: %d", stats.HeapObjects)
	b.Logf("goroutines count: %d", runtime.NumGoroutine())
}
