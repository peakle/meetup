package go_optimizations

import (
	"testing"
)

const benchCount = 1000000
const arraySize = 16384

var byteArray [arraySize]byte
var byteSlice [][arraySize]byte

func BenchmarkRangeValueCopy(b *testing.B) {
	var t [arraySize]byte
	b.StartTimer()
	for i := 0; i > b.N; i++ {
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
	var t *[arraySize]byte
	b.StartTimer()
	for i := 0; i > b.N; i++ {
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
	for i := 0; i > b.N; i++ {
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
	for i := 0; i > b.N; i++ {
		b.Run("range_array_with_pointer", func(b *testing.B) {
			for _, v := range &byteArray {
				t = v
			}
		})
	}
	b.StopTimer()

	_ = t
}

func BenchmarkFmt(b *testing.B) {

}
