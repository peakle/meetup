package go_optimizations

import (
	"context"
	customFmt "go-optimizations/fmt"
	"runtime"
	"sync"
	"testing"
	"time"

	pool "github.com/delivery-club/bees"
)

const (
	benchCount = 1000000
	poolSize   = 500000
)

const (
	extraSmallArraySize = 64 << (1 * iota)
	smallArraySize
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

var byteArray [hugeArraySize]byte
var byteSlice [][hugeArraySize]byte

func BenchmarkRangeValueCopy(b *testing.B) {
	b.StopTimer()
	var t [hugeArraySize]byte
	b.StartTimer()

	b.Run("range_value_copy", func(b *testing.B) {
		for _, str := range byteSlice {
			t = str
		}
	})
	_ = t
}

func BenchmarkRangeValueIndex(b *testing.B) {
	b.StopTimer()
	var t *[hugeArraySize]byte
	b.StartTimer()

	b.Run("range_value_index", func(b *testing.B) {
		for ii := range byteSlice {
			t = &byteSlice[ii]
		}
	})
	_ = t
}

func BenchmarkRangeArrayValue(b *testing.B) {
	b.StopTimer()
	var t byte
	b.StartTimer()

	b.Run("range_array", func(b *testing.B) {
		for _, v := range byteArray {
			t = v
		}
	})
	_ = t
}

func BenchmarkRangeArrayWithPointer(b *testing.B) {
	b.StopTimer()
	var t byte
	b.StartTimer()

	b.Run("range_array_with_pointer", func(b *testing.B) {
		for _, v := range &byteArray {
			t = v
		}
	})
	_ = t
}

func BenchmarkMakeIncorrectUsage(b *testing.B) {
	b.StopTimer()
	var t = make([][extraSmallArraySize]byte, 10)
	b.StartTimer()

	b.Run("benchmark_make_incorrect_usage", func(b *testing.B) {
		for ii := 0; ii < benchCount; ii++ {
			t = append(t, [extraSmallArraySize]byte{})
		}
	})
}

func BenchmarkMakeCorrectUsage(b *testing.B) {
	b.StopTimer()
	var t = make([][extraSmallArraySize]byte, 0, benchCount)

	b.StartTimer()

	b.Run("benchmark_make_correct_usage", func(b *testing.B) {
		for ii := 0; ii < benchCount; ii++ {
			t = append(t, [extraSmallArraySize]byte{})
		}
	})
}

type hugeStruct struct {
	h     uint64
	cache [hugeArraySize]byte
	body  []byte
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
}

var hugeStructPool sync.Pool

func BenchmarkNewObjectWithSyncPool(b *testing.B) {
	b.StopTimer()
	get := func() *hugeStruct {
		h := hugeStructPool.Get()
		if h == nil {
			return &hugeStruct{body: make([]byte, 0, mediumArraySize)}
		}
		return h.(*hugeStruct)
	}
	put := func(h *hugeStruct) {
		h.h = 0
		h.body = h.body[:0]
		hugeStructPool.Put(h)
	}

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
}

func dummyProcess(h *hugeStruct) uint64 {
	for i := 0; i < 1000; i++ {
		h.h = uint64(i)
	}
	time.Sleep(time.Microsecond)
	return h.h
}

func BenchmarkGoroutinesRaw(b *testing.B) {
	b.StopTimer()
	var (
		wg sync.WaitGroup
		h  = &hugeStruct{}
	)
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		wg.Add(benchCount)
		for j := 0; j < benchCount; j++ {
			go func() {
				dummyProcess(h)
				wg.Done()
			}()
		}
	}
	wg.Wait()
	b.StopTimer()

	b.Logf("memory usage:%d MB", checkMem())
}

func BenchmarkGoroutinesSemaphore(b *testing.B) {
	b.StopTimer()
	var (
		wg sync.WaitGroup
		h  = &hugeStruct{}
	)
	sema := make(chan struct{}, poolSize)

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		wg.Add(benchCount)
		for j := 0; j < benchCount; j++ {
			sema <- struct{}{}
			go func() {
				dummyProcess(h)
				<-sema
				wg.Done()
			}()
		}
	}
	wg.Wait()
	b.StopTimer()

	b.Logf("memory usage:%d MB", checkMem())
}

func BenchmarkReusableGoroutines(b *testing.B) {
	b.StopTimer()
	var (
		wg sync.WaitGroup
		h  = &hugeStruct{}
	)

	p := pool.Create(context.Background(), func(ctx context.Context, task interface{}) {
		dummyProcess(h)
		wg.Done()
	}, pool.WithCapacity(poolSize), pool.WithKeepAlive(5*time.Second))
	defer func() {
		p.Close()
	}()
	var task interface{}

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		wg.Add(benchCount)
		for j := 0; j < benchCount; j++ {
			p.Submit(task)
		}
	}
	wg.Wait()
	b.StopTimer()

	b.Logf("memory usage:%d MB", checkMem())
}

func BenchmarkGC(b *testing.B)            {}
func BenchmarkGCWithBallast(b *testing.B) {}
func BenchmarkStrings(b *testing.B)       {}
func BenchmarkBytes(b *testing.B)         {}

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

	// TODO why slower than fmt.Sprint
	b.Run("fmt_sprint_string", func(b *testing.B) {
		for i := 0; i < benchCount; i++ {
			foo = customFmt.SprintString("foo", "bar")
		}
	})

	b.Run("fmt_sprint", func(b *testing.B) {
		for i := 0; i < benchCount; i++ {
			foo = customFmt.Sprint("foo", "bar")
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

func BenchmarkStructSizes(b *testing.B) {}

func checkMem() uint64 {
	var curMem uint64
	mem := runtime.MemStats{}
	runtime.ReadMemStats(&mem)
	curMem = mem.TotalAlloc/MiB - curMem
	return curMem
}
