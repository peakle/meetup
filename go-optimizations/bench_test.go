package go_optimizations

import (
	"context"
	"fmt"
	customFmt "go-optimizations/fmt"
	"runtime"
	"sync"
	"testing"
	"time"
	"unsafe"

	pool "github.com/delivery-club/bees"
)

const (
	benchCount = 1000000
	poolSize   = 500000
)

const (
	extraSmallArraySize = 64 << (1 * iota)
	_
	_
	_
	mediumArraySize
	hugeArraySize
)

const (
	_ = 1 << (10 * iota)
	_
	MiB
)

type hugeStruct struct {
	h     uint64
	cache [hugeArraySize]byte
	body  []byte
}

var hugeArray = [hugeArraySize]hugeStruct{}

func BenchmarkRangeValueCopy(b *testing.B) {
	var sum uint64 = 0
	var hugeSlice = make([]hugeStruct, benchCount)

	b.Run("range_value_copy", func(b *testing.B) {
		for _, hs := range hugeSlice {
			sum += hs.h
		}
	})

	b.Run("range_value_index", func(b *testing.B) {
		for ii := range hugeSlice {
			sum += hugeSlice[ii].h
		}
	})

	b.Run("range_value_pointer_and_index", func(b *testing.B) {
		for ii := range hugeSlice {
			sum = (&hugeSlice[ii]).h
		}
	})

	_ = sum
}

func BenchmarkRangeArrayValue(b *testing.B) {
	b.StopTimer()
	var sum uint64 = 0
	b.StartTimer()

	b.Run("range_array", func(b *testing.B) {
		for _, v := range hugeArray {
			sum += v.h
		}
	})

	_ = sum
}

func BenchmarkRangeArrayWithPointer(b *testing.B) {
	b.StopTimer()
	var sum uint64 = 0
	b.StartTimer()

	b.Run("range_array_with_pointer", func(b *testing.B) {
		for _, v := range &hugeArray {
			sum += v.h
		}
	})

	_ = sum
}

func BenchmarkMakeIncorrectUsage(b *testing.B) {
	b.Run("benchmark_make_incorrect_usage", func(b *testing.B) {
		var t = make([][extraSmallArraySize]byte, 0, 10)

		for ii := 0; ii < benchCount; ii++ {
			t = append(t, [extraSmallArraySize]byte{})
		}
	})
}

func BenchmarkMakeCorrectUsage(b *testing.B) {
	b.Run("benchmark_make_correct_usage", func(b *testing.B) {
		var t = make([][extraSmallArraySize]byte, 0, benchCount)

		for ii := 0; ii < benchCount; ii++ {
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
		for ii := 0; ii < benchCount; ii++ {
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
		for ii := 0; ii < benchCount; ii++ {
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
	b.StartTimer()

	b.Run("new_object", func(b *testing.B) {
		var wg sync.WaitGroup
		wg.Add(benchCount)

		for ii := 0; ii < benchCount; ii++ {
			go func() {
				h := &hugeStruct{body: make([]byte, 0, mediumArraySize)}
				h = dummyPointer(h)
				wg.Done()
			}()
		}

		wg.Wait()
	})

	stats := checkMem()
	b.Logf("memory usage:%d MB", stats.TotalAlloc/MiB)
	b.Logf("GC cycles: %d", stats.NumGC)
}

var hugeStructPool sync.Pool

func get() *hugeStruct {
	h := hugeStructPool.Get()
	if h == nil {
		return &hugeStruct{body: make([]byte, 0, mediumArraySize)}
	}
	return h.(*hugeStruct)
}

func put(h *hugeStruct) {
	h.h = 0
	h.body = h.body[:0]
	hugeStructPool.Put(h)
}

func BenchmarkNewObjectWithSyncPool(b *testing.B) {
	b.StopTimer()
	b.StartTimer()
	b.Run("new_object_with_sync_pool", func(b *testing.B) {
		var wg sync.WaitGroup
		wg.Add(benchCount)

		for ii := 0; ii < benchCount; ii++ {
			go func() {
				h := get()
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
}

func dummyProcess() uint64 {
	var sum uint64
	for i := 0; i < 1000; i++ {
		sum += uint64(i)
	}
	time.Sleep(time.Microsecond)
	return sum
}

func BenchmarkGoroutinesRaw(b *testing.B) {
	b.StopTimer()
	var wg sync.WaitGroup
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		wg.Add(benchCount)
		for j := 0; j < benchCount; j++ {
			go func() {
				dummyProcess()
				wg.Done()
			}()
		}
	}
	wg.Wait()
	b.StopTimer()

	stats := checkMem()
	b.Logf("memory usage:%d MB", stats.TotalAlloc/MiB)
	b.Logf("GC cycles: %d", stats.NumGC)
}

func BenchmarkGoroutinesSemaphore(b *testing.B) {
	b.StopTimer()
	var wg sync.WaitGroup
	sema := make(chan struct{}, poolSize)

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		wg.Add(benchCount)
		for j := 0; j < benchCount; j++ {
			sema <- struct{}{}
			go func() {
				dummyProcess()
				<-sema
				wg.Done()
			}()
		}
	}
	wg.Wait()
	b.StopTimer()

	stats := checkMem()
	b.Logf("memory usage:%d MB", stats.TotalAlloc/MiB)
	b.Logf("GC cycles: %d", stats.NumGC)
}

func BenchmarkReusableGoroutines(b *testing.B) {
	b.StopTimer()
	var wg sync.WaitGroup

	p := pool.Create(context.Background(), func(ctx context.Context, task interface{}) {
		dummyProcess()
		wg.Done()
	}, pool.WithCapacity(poolSize), pool.WithKeepAlive(5*time.Second))
	defer func() {
		p.Close()
	}()

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		wg.Add(benchCount)
		for j := 0; j < benchCount; j++ {
			p.Submit(nil)
		}
	}
	wg.Wait()
	b.StopTimer()

	stats := checkMem()
	b.Logf("memory usage:%d MB", stats.TotalAlloc/MiB)
	b.Logf("GC cycles: %d", stats.NumGC)
}

func BenchmarkGC(b *testing.B) {
	b.StopTimer()
	var wg sync.WaitGroup
	wg.Add(1)
	b.StartTimer()

	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			dummyApplication(benchCount / 2)
			time.Sleep(time.Nanosecond)
		}
	}()
	for i := 0; i < 10; i++ {
		dummyApplication(benchCount)
		time.Sleep(time.Nanosecond)
	}
	wg.Wait()
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
	wg.Add(1)
	b.StartTimer()

	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			dummyApplication(benchCount / 2)
			time.Sleep(time.Nanosecond)
		}
	}()
	for i := 0; i < 10; i++ {
		dummyApplication(benchCount)
		time.Sleep(time.Nanosecond)
	}
	wg.Wait()
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

func BenchmarkStrings(b *testing.B) {}
func BenchmarkBytes(b *testing.B)   {}

func BenchmarkInterfaceUsage(b *testing.B) {
	b.StopTimer()
	var (
		h = &hugeStruct{
			h:     0,
			cache: [2048]byte{},
			body:  make([]byte, hugeArraySize),
		}
		foo string
	)
	b.StartTimer()

	b.Run("fmt_sprintf", func(b *testing.B) {
		for i := 0; i < benchCount; i++ {
			foo = fmt.Sprintf("foo bar")
		}
	})

	b.Run("fmt_sprint", func(b *testing.B) {
		for i := 0; i < benchCount; i++ {
			foo = fmt.Sprint("foo bar")
		}
	})

	// TODO why slower than fmt.Sprint
	b.Run("fmt_sprint_string", func(b *testing.B) {
		for i := 0; i < benchCount; i++ {
			foo = customFmt.SprintString("foo", "bar")
		}
	})

	b.Run("concatenation", func(b *testing.B) {
		for i := 0; i < benchCount; i++ {
			foo = "foo" + "bar"
		}
	})

	foo = "bar"
	_ = foo

	dummyPointer(h)
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
}

func checkMem() *runtime.MemStats {
	mem := &runtime.MemStats{}
	runtime.ReadMemStats(mem)
	return mem
}
