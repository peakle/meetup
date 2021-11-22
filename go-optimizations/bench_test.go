package go_optimizations

import (
	"context"
	"fmt"
	customFmt "go-optimizations/fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"unsafe"

	pool "github.com/delivery-club/bees"
)

//Как запускать бенчмарки:
// go test --bench=BenchmarkNewObject$ --benchmem -v --count=10
// go test --bench=. --benchmem -v --count=10
// count - bench run count
// benctime - count of iterations b.N
// benchmem - report allocs

const (
	benchCount = 1000000
	poolSize   = 500000
)

const (
	extraSmallArraySize = 64 << (1 * iota)
	smallArraySize
	averageArraySize
	mediumArraySize
	bigArraySize
	hugeArraySize
)

const (
	_ = 1 << (10 * iota)
	_
	MiB
)

type (
	extraSmallStruct struct {
		h     uint64
		cache [extraSmallArraySize]byte
		body  []byte
	}
	smallStruct struct {
		h     uint64
		cache [smallArraySize]byte
		body  []byte
	}
	averageStruct struct {
		h     uint64
		cache [averageArraySize]byte
		body  []byte
	}
	mediumStruct struct {
		h     uint64
		cache [mediumArraySize]byte
		body  []byte
	}
	bigStruct struct {
		h     uint64
		cache [bigArraySize]byte
		body  []byte
	}
	hugeStruct struct {
		h     uint64
		cache [hugeArraySize]byte
		body  []byte
	}
)

var hugeStructPool = sync.Pool{New: func() interface{} {
	return &hugeStruct{body: make([]byte, 0, mediumArraySize)}
}}

func BenchmarkRangeValueCopy(b *testing.B) {
	var sum uint64 = 0
	var hugeSlice = make([]hugeStruct, benchCount)

	b.Run("range_value_copy", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, hs := range hugeSlice {
				sum += hs.h
			}
		}
	})

	b.Run("range_value_index", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for ii := range hugeSlice {
				sum += hugeSlice[ii].h
			}
		}
	})

	b.Run("range_value_pointer_and_index", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for ii := range hugeSlice {
				sum += (&hugeSlice[ii]).h
			}
		}
	})

	b.Run("range_value_pointer_and_index_split", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for ii := range hugeSlice {
				h := &hugeSlice[ii]
				sum += h.h
			}
		}
	})

	b.Logf("sum: %d", sum)
	stats := checkMem()
	b.Logf("memory usage:%d MB", stats.TotalAlloc/MiB)
	b.Logf("GC cycles: %d", stats.NumGC)
}

func BenchmarkRangeArrayValue(b *testing.B) {
	b.StopTimer()
	var sum uint64 = 0
	var hugeArray = [hugeArraySize]hugeStruct{}
	b.StartTimer()

	b.Run("range_array", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, v := range hugeArray {
				sum += v.h
			}
		}
	})

	b.Run("range_array_with_pointer", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, v := range &hugeArray {
				sum += v.h
			}
		}
	})

	_ = sum
}

func BenchmarkMakeIncorrectUsage(b *testing.B) {
	b.Run("benchmark_make_incorrect_usage", func(b *testing.B) {
		var t = make([][extraSmallArraySize]byte, 0, 10)

		for ii := 0; ii < b.N; ii++ {
			t = append(t, [extraSmallArraySize]byte{})
		}
	})
}

func BenchmarkMakeCorrectUsage(b *testing.B) {
	b.Run("benchmark_make_correct_usage", func(b *testing.B) {
		var t = make([][extraSmallArraySize]byte, 0, b.N)

		for ii := 0; ii < b.N; ii++ {
			t = append(t, [extraSmallArraySize]byte{})
		}
	})
}

func BenchmarkHugeParamByCopy(b *testing.B) {
	b.StopTimer()
	t := hugeStruct{
		h:     0,
		cache: [2048]byte{},
	}

	b.StartTimer()

	b.Run("benchmark_huge_param_by_copy", func(b *testing.B) {
		for ii := 0; ii < b.N; ii++ {
			t = dummyCopy(t)
		}
	})
}

func dummyCopy(h hugeStruct) hugeStruct {
	for i := 0; i < 10; i++ {
		h.h = uint64(i)
	}

	return h
}

func BenchmarkHugeParamByPointer(b *testing.B) {
	b.StopTimer()
	t := &hugeStruct{
		h:     0,
		cache: [2048]byte{},
	}

	b.StartTimer()
	b.Run("benchmark_huge_param_by_pointer", func(b *testing.B) {
		for ii := 0; ii < b.N; ii++ {
			t = dummyPointer(t)
		}
	})
}

func dummyPointer(h *hugeStruct) *hugeStruct {
	for i := 0; i < 10; i++ {
		h.h = uint64(i)
		h.body = append(h.body, 'f')
	}

	return h
}

func BenchmarkNewObject(b *testing.B) {
	b.StopTimer()
	var (
		wg sync.WaitGroup
		h  *hugeStruct
	)
	b.StartTimer()

	b.Run("new_object", func(b *testing.B) {
		wg.Add(b.N)

		for ii := 0; ii < b.N; ii++ {
			go func() {
				h = &hugeStruct{body: make([]byte, 0, mediumArraySize)}
				h = dummyPointer(h)
				wg.Done()
			}()
		}

		wg.Wait()
	})

	stats := checkMem()
	b.Logf("memory usage:%d MB", stats.TotalAlloc/MiB)
	b.Logf("GC cycles: %d", stats.NumGC)
	b.Logf("counter: %d", h.h)
}

func get() *hugeStruct {
	return hugeStructPool.Get().(*hugeStruct)
}

func put(h *hugeStruct) {
	h.h = 0
	h.body = h.body[:0]
	for i := range &h.cache {
		h.cache[i] = 0
	}
	hugeStructPool.Put(h)
}

func BenchmarkNewObjectWithSyncPool(b *testing.B) {
	b.StopTimer()
	hugeStructPool = sync.Pool{New: func() interface{} {
		return &hugeStruct{body: make([]byte, 0, mediumArraySize)}
	}}
	var h *hugeStruct
	b.StartTimer()
	b.Run("new_object_with_sync_pool", func(b *testing.B) {
		var wg sync.WaitGroup
		wg.Add(b.N)

		for ii := 0; ii < b.N; ii++ {
			go func() {
				h = get()
				h = dummyPointer(h)
				wg.Done()

				put(h)
			}()
		}

		wg.Wait()
	})

	stats := checkMem()
	b.Logf("memory usage:%d MB", stats.TotalAlloc/MiB)
	b.Logf("GC cycles: %d", stats.NumGC)
	b.Logf("counter: %d", h.h)
}

var mu sync.Mutex

func dummyProcess(rand int64) int64 {
	var sum int64
	for i := int64(0); i < 1000; i++ {
		sum += i
	}

	mu.Lock()
	sum += rand
	mu.Unlock()

	return sum
}

func BenchmarkGoroutinesRaw(b *testing.B) {
	b.StopTimer()
	var (
		wg      sync.WaitGroup
		counter int64
		process = func(num int64) {
			atomic.AddInt64(&counter, dummyProcess(num))
			wg.Done()
		}
	)
	b.StartTimer()

	b.Run("raw_goroutines", func(b *testing.B) {
		wg.Add(b.N)
		for j := 0; j < b.N; j++ {
			go process(int64(j))
		}
		wg.Wait()
	})
	b.StopTimer()

	stats := checkMem()
	b.Logf("memory usage:%d MB", stats.TotalAlloc/MiB)
	b.Logf("GC cycles: %d", stats.NumGC)
	b.Logf("%d", counter)
}

func BenchmarkGoroutinesRawNotOptimized(b *testing.B) {
	b.StopTimer()
	var (
		wg      sync.WaitGroup
		counter int64
	)
	b.StartTimer()

	b.Run("raw_goroutines_not_optimized", func(b *testing.B) {
		wg.Add(b.N)
		for j := 0; j < b.N; j++ {
			go func(num int64) {
				atomic.AddInt64(&counter, dummyProcess(num))
				wg.Done()
			}(int64(j))
		}
		wg.Wait()
	})

	stats := checkMem()
	b.Logf("memory usage:%d MB", stats.TotalAlloc/MiB)
	b.Logf("GC cycles: %d", stats.NumGC)
	b.Logf("%d", counter)
}

func BenchmarkGoroutinesSemaphore(b *testing.B) {
	b.StopTimer()
	var (
		wg      sync.WaitGroup
		counter int64
		sema    = make(chan struct{}, poolSize)
		process = func(num int64) {
			atomic.AddInt64(&counter, dummyProcess(num))
			<-sema
			wg.Done()
		}
	)
	b.StartTimer()

	b.Run("semaphore", func(b *testing.B) {
		wg.Add(b.N)
		for j := 0; j < b.N; j++ {
			sema <- struct{}{}
			go process(int64(j))
		}
		wg.Wait()
	})
	b.StopTimer()

	stats := checkMem()
	b.Logf("memory usage:%d MB", stats.TotalAlloc/MiB)
	b.Logf("GC cycles: %d", stats.NumGC)
	b.Logf("%d", counter)
}

func BenchmarkGoroutinesReusable(b *testing.B) {
	b.StopTimer()
	var (
		wg      sync.WaitGroup
		counter int64
	)

	p := pool.Create(context.Background(), func(ctx context.Context, task interface{}) {
		atomic.AddInt64(&counter, dummyProcess(task.(int64)))
		wg.Done()
	}, pool.WithCapacity(poolSize), pool.WithKeepAlive(5*time.Second))
	defer func() {
		p.Close()
	}()

	b.StartTimer()

	b.Run("reusable_goroutines", func(b *testing.B) {
		wg.Add(b.N)
		for j := 0; j < b.N; j++ {
			p.Submit(int64(j))
		}
		wg.Wait()
	})
	b.StopTimer()

	stats := checkMem()
	b.Logf("memory usage:%d MB", stats.TotalAlloc/MiB)
	b.Logf("GC cycles: %d", stats.NumGC)
	b.Logf("%d", counter)
}

func BenchmarkGC(b *testing.B) {
	b.StopTimer()
	var wg sync.WaitGroup
	b.StartTimer()

	b.Run("gc_without_ballast", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				dummyApplication(benchCount / 2)
				time.Sleep(time.Nanosecond)
			}()
			dummyApplication(benchCount)
			time.Sleep(time.Nanosecond)
			wg.Wait()
		}
	})
	b.StopTimer()

	stats := checkMem()
	b.Logf("memory usage: %d MB", stats.TotalAlloc/MiB)
	b.Logf("GC cycles: %d", stats.NumGC)
}

//bench_test.go:329: memory usage: 59522 MB
//bench_test.go:330: GC cycles: 17838

func BenchmarkGCWithBallast(b *testing.B) {
	b.StopTimer()
	var (
		ballast = make([]byte, 10<<30)
		wg      sync.WaitGroup
	)
	b.StartTimer()

	b.Run("gc_with_ballast", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				dummyApplication(benchCount / 2)
				time.Sleep(time.Nanosecond)
			}()
			dummyApplication(benchCount)
			time.Sleep(time.Nanosecond)
			wg.Wait()
		}
	})
	b.StopTimer()

	func(_ []byte) {
		for {
			break
		}
	}(ballast)
	_ = ballast
	stats := checkMem()
	b.Logf("memory usage: %d MB", stats.TotalAlloc/MiB)
	b.Logf("GC cycles: %d", stats.NumGC)
	b.Logf("%d", ballast[2])
}

//bench_test.go:353: memory usage: 69813 MB
//bench_test.go:354: GC cycles: 17017

func dummyApplication(count int) {
	var wg sync.WaitGroup
	wg.Add(count)

	for ii := 0; ii < count; ii++ {
		go func() {
			h := &hugeStruct{body: make([]byte, 0, mediumArraySize)}
			h = dummyPointer(h)
			wg.Done()
		}()
	}
	wg.Wait()
}

func BenchmarkInterfaceUsage(b *testing.B) {
	b.StopTimer()
	var foo string
	b.StartTimer()

	b.Run("fmt_sprint", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			foo = fmt.Sprint("foo", "bar")
		}
	})

	b.Run("fmt_sprintf", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			foo = fmt.Sprintf("foo %s", "bar")
		}
	})

	// TODO why slower than fmt.Sprint
	b.Run("fmt_sprint_string", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			foo = customFmt.SprintString("foo", "bar")
		}
	})

	b.Run("concatenation", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			foo = "foo" + "bar"
		}
	})

	foo = "bar"
	_ = foo
}

func BenchmarkStructSizes(b *testing.B) {
	type struct1 struct {
		counter       int8
		secondCounter int8
		k             chan string
	}

	type struct2 struct {
		counter       int8
		k             chan string
		secondCounter int8
	}

	b.Logf("%d", unsafe.Sizeof(struct1{}))
	b.Logf("%d", unsafe.Sizeof(struct2{}))
	b.Logf("%d", unsafe.Sizeof(extraSmallStruct{}))
	b.Logf("%d", unsafe.Sizeof(smallStruct{}))
	b.Logf("%d", unsafe.Sizeof(averageStruct{}))
	b.Logf("%d", unsafe.Sizeof(mediumStruct{}))
	b.Logf("%d", unsafe.Sizeof(bigStruct{}))
	b.Logf("%d", unsafe.Sizeof(hugeStruct{}))
}

type state struct {
	c int64
}

const num = 1000000

func BenchmarkAtomicBased(b *testing.B) {
	var (
		counter = &state{}
		wg      sync.WaitGroup
	)

	atomicCounter := func(i int64) int64 {
		var taken bool
		var newCounter int64

		for !taken {
			oldCounter := atomic.LoadInt64(&counter.c)
			newCounter = i * oldCounter

			taken = atomic.CompareAndSwapInt64(&counter.c, oldCounter, newCounter)
		}
		return newCounter
	}

	var trigger int64

	process := func(i int64) {
		for trigger == 0 {
		}
		atomicCounter(i)
		wg.Done()
	}

	b.Run("atomic_based_counter", func(b *testing.B) {
		wg.Add(b.N)
		for i := 0; i < b.N; i++ {
			go process(int64(i))
		}
		atomic.StoreInt64(&trigger, 1)
		wg.Wait()
	})

	b.Logf("counter: %d", counter.c)
}

func BenchmarkMutexBased(b *testing.B) {
	var (
		counter = &state{}
		wg      sync.WaitGroup
		mu      sync.Mutex
	)

	mutexCounter := func(i int64) int64 {
		mu.Lock()
		newCounter := i * counter.c
		counter.c = newCounter
		mu.Unlock()

		return newCounter
	}
	var trigger int64

	process := func(i int64) {
		for trigger == 0 {
		}
		mutexCounter(i)
		wg.Done()
	}

	b.Run("mutex_based_counter", func(b *testing.B) {
		wg.Add(b.N)
		for i := 0; i < b.N; i++ {
			go process(int64(i))
		}
		atomic.StoreInt64(&trigger, 1)
		wg.Wait()
	})

	b.Logf("counter: %d", counter.c)
}

func checkMem() *runtime.MemStats {
	mem := &runtime.MemStats{}
	runtime.ReadMemStats(mem)
	return mem
}
