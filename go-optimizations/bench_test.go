package go_optimizations

import (
	"testing"
)

const benchCount = 1000000

const (
	smallArraySize = 1024 << (1 * iota)
	mediumArraySize
	hugeArraySize
)

var byteArray [hugeArraySize]byte
var byteSlice [][hugeArraySize]byte

func BenchmarkRangeValueCopy(b *testing.B) {
	var t [hugeArraySize]byte
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		b.Run("range_value_copy", func(b *testing.B) {
			for _, str := range byteSlice {
				t = str
			}
		})
	}
	b.StopTimer()

	_ = t
}

func BenchmarkRangeValueIndex(b *testing.B) {
	var t *[hugeArraySize]byte
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		b.Run("range_value_index", func(b *testing.B) {
			for ii := range byteSlice {
				t = &byteSlice[ii]
			}
		})
	}
	b.StopTimer()

	_ = t
}

func BenchmarkRangeArrayValue(b *testing.B) {
	var t byte
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		b.Run("range_array", func(b *testing.B) {
			for _, v := range byteArray {
				t = v
			}
		})
	}
	b.StopTimer()

	_ = t
}

func BenchmarkRangeArrayWithPointer(b *testing.B) {
	var t byte
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		b.Run("range_array_with_pointer", func(b *testing.B) {
			for _, v := range &byteArray {
				t = v
			}
		})
	}
	b.StopTimer()

	_ = t
}

func BenchmarkMakeIncorrectUsage(b *testing.B) {
	var t = make([][smallArraySize]byte, 10)

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		b.Run("benchmark_make_uncorrect_usage", func(b *testing.B) {
			for ii := 0; ii < benchCount; ii++ {
				t = append(t, [smallArraySize]byte{})
			}
		})
	}
	b.StopTimer()
}

func BenchmarkMakeCorrectUsage(b *testing.B) {
	var t = make([][smallArraySize]byte, 0, benchCount)

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		b.Run("benchmark_make_correct_usage", func(b *testing.B) {
			for ii := 0; ii < benchCount; ii++ {
				t = append(t, [smallArraySize]byte{})
			}
		})
	}
	b.StopTimer()
}
