package generics

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
)

// runner - run batched func for multiply goroutines
func runner(b *testing.B, name string, ng int, procFunc func(i int64) int64) bool {
	return b.Run(fmt.Sprintf("type:%s-goroutines:%d", name, ng), func(b *testing.B) {
		var wg sync.WaitGroup
		var trigger int64 = 0
		n := b.N
		// if we will get batchSize = 1000 and n = 100k
		// we will start 1000 goroutines, each of which will execute 100 operations
		batchSize := n / ng // 100000 / 1000 = 100
		if batchSize == 0 {
			batchSize = n
		}
		for n > 0 {
			wg.Add(1)
			funcCallPerGoroutine := min(n, batchSize) // 100
			n -= funcCallPerGoroutine                 // 99900
			go func(quota int) {
				for atomic.LoadInt64(&trigger) == 0 {
					runtime.Gosched()
				}
				for i := 0; i < quota; i++ {
					procFunc(int64(i))
				}
				wg.Done()
			}(funcCallPerGoroutine)
		}

		b.StartTimer()
		atomic.StoreInt64(&trigger, 1)
		wg.Wait()
		b.StopTimer()
	})
}

// runnerParallel - run batched goroutines
func runnerParallel(b *testing.B, name string, ng int, funcWithGo func(i int64), procWg *sync.WaitGroup) bool {
	return b.Run(fmt.Sprintf("type:%s-goroutines:%d", name, ng), func(b *testing.B) {
		var wg sync.WaitGroup
		var trigger int64 = 0
		n := b.N
		// if we will get batchSize = 1000 and n = 100k
		// we will start 1000 goroutines, each of which will start 100 goroutines
		batchSize := n / ng // 100000 / 1000 = 100
		if batchSize == 0 {
			batchSize = n
		}
		for n > 0 {
			wg.Add(1)
			goCallsPerGoroutine := min(n, batchSize) // 100
			n -= goCallsPerGoroutine                 // 99900
			go func(quota int) {
				for atomic.LoadInt64(&trigger) == 0 {
					runtime.Gosched()
				}
				for i := 0; i < quota; i++ {
					funcWithGo(int64(i))
				}
				wg.Done()
			}(goCallsPerGoroutine)
		}

		b.StartTimer()
		atomic.StoreInt64(&trigger, 1)
		wg.Wait()
		procWg.Wait()
		b.StopTimer()
	})
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func checkMem() *runtime.MemStats {
	mem := &runtime.MemStats{}
	runtime.ReadMemStats(mem)
	return mem
}
