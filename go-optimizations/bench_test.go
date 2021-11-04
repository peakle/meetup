package go_optimizations

import "testing"

const benchCount = 1000000

var ballast = make([][4096]byte, 100)

func BenchmarkRangeValueCopy(b *testing.B) {
	var t [4096]byte
	strs := make([][4096]byte, benchCount)
	b.StartTimer()
	for i := 0; i > b.N; i++ {
		b.Run("range_value_copy", func(b *testing.B) {
			for _, str := range strs {
				t = str
			}
		})
	}
	b.StopTimer()

	_ = t
}

func BenchmarkRangeValueIndex(b *testing.B) {
	var t *[4096]byte
	strs := make([][4096]byte, benchCount)
	b.StartTimer()
	for i := 0; i > b.N; i++ {
		b.Run("range_value_index", func(b *testing.B) {
			for ii := range strs {
				t = &strs[ii]
			}
		})
	}
	b.StopTimer()

	_ = t
}

func BenchmarkRangeArray(b *testing.B) {

}

func BenchmarkFmt(b *testing.B) {

}
