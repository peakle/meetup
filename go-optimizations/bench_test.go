package go_optimizations

import (
	customFmt "go-optimizations/fmt"
	"sync"
	"testing"
)

const benchCount = 1000000

const (
	extraSmallArraySize = 64 << (1 * iota)
	smallArraySize
	_
	_
	mediumArraySize
	hugeArraySize
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
	h     int
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
		h.h = i
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
		h.h = i
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

func BenchmarkSlice(b *testing.B) {

}

func BenchmarkSliceReuse(b *testing.B) {

}

func BenchmarkRawGoroutines(b *testing.B)       {}
func BenchmarkSemaphoreGoroutines(b *testing.B) {}
func BenchmarkReusableGoroutines(b *testing.B)  {}
func BenchmarkGC(b *testing.B)                  {}
func BenchmarkGCWithBallast(b *testing.B)       {}
func BenchmarkStrings(b *testing.B)             {}
func BenchmarkBytes(b *testing.B)               {}

func BenchmarkInterfaceUsage(b *testing.B) {
	b.StopTimer()
	h := &hugeStruct{
		h:     0,
		cache: [2048]byte{},
		body:  make([]byte, hugeArraySize),
	}
	b.StartTimer()

	var foo string
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
