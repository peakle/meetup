package generics

import (
	"runtime"
	"testing"
)

const (
	_ = 1 << (10 * iota)
	_
	MiB
)

type (
	Foo struct {
		i int // struct without params will be optimized us unused
	}
	Bar struct {
		i int
	}
)

type (
	StackGenerics[T any] []T
	StackInterface       []interface{}
	StackFoo             []Foo
)

////// Methods for StackGenerics type //////////////
func (s StackGenerics[T]) Peek() T {
	return s[len(s)-1]
}

func (s *StackGenerics[T]) Pop() {
	*s = (*s)[:len(*s)-1]
}

func (s *StackGenerics[T]) Push(value T) {
	*s = append(*s, value)
}

////// Methods for StackInterface type //////////////
func (s StackInterface) Peek() interface{} {
	return s[len(s)-1]
}

func (s *StackInterface) Pop() {
	*s = (*s)[:len(*s)-1]
}

func (s *StackInterface) Push(value interface{}) {
	*s = append(*s, value)
}

////// Methods for StackFoo type //////////////
func (s StackFoo) Peek() Foo {
	return s[len(s)-1]
}

func (s *StackFoo) Pop() {
	*s = (*s)[:len(*s)-1]
}

func (s *StackFoo) Push(value Foo) {
	*s = append(*s, value)
}

////// Generic function ////////////////////

func Peek[T any](s []T) T {
	return s[len(s)-1]
}

func  Pop[T any](s []T) []T{
	s = s[:len(s)-1]
	return s
}

func  Push[T any](s []T, value T)[]T {
	s = append(s, value)
	return s
}
////// Global vars for disable compiler optimizations for loops //////
var (
	foo   Foo
	bar   Bar
)

// go test -test.bench=BenchmarkStackInterface -count=3 -test.benchtime=100000000x
func BenchmarkStackInterface(b *testing.B) {
	var s StackInterface
	for i := 0; i < b.N; i++ {
		s.Push(Foo{})
		s.Push(Foo{})
		s.Pop()
		foo = s.Peek().(Foo) // type assertion that cannot be optimized as unusable var, so we no need to use sink var
	}

	b.StopTimer()
	stats := checkMem()
	b.Logf("memory usage:%d MB", stats.TotalAlloc/MiB)
	b.Logf("GC cycles: %d", stats.NumGC)
}

func BenchmarkStackGenericsFoo(b *testing.B) {
	var s StackGenerics[Foo]
	for i := 0; i < b.N; i++ {
		s.Push(Foo{})
		s.Push(Foo{})
		s.Pop()
		foo = s.Peek()
	}

	b.StopTimer()
	stats := checkMem()
	b.Logf("memory usage:%d MB", stats.TotalAlloc/MiB)
	b.Logf("GC cycles: %d", stats.NumGC)
}

func BenchmarkStackGenericsBar(b *testing.B) {
	var s StackGenerics[Bar]
	for i := 0; i < b.N; i++ {
		s.Push(Bar{})
		s.Push(Bar{})
		s.Pop()
		bar = s.Peek()
	}

	b.StopTimer()
	stats := checkMem()
	b.Logf("memory usage:%d MB", stats.TotalAlloc/MiB)
	b.Logf("GC cycles: %d", stats.NumGC)
}

func BenchmarkStackTyped(b *testing.B) {
	var s StackFoo
	for i := 0; i < b.N; i++ {
		s.Push(Foo{})
		s.Push(Foo{})
		s.Pop()
		foo = s.Peek()
	}

	b.StopTimer()
	stats := checkMem()
	b.Logf("memory usage:%d MB", stats.TotalAlloc/MiB)
	b.Logf("GC cycles: %d", stats.NumGC)
}

//go:noinnline
func BenchmarkStackGenericFunc(b *testing.B) {
	var s StackFoo
	for i := 0; i < b.N; i++ {
		s = Push(s, Foo{})
		s = Push(s, Foo{})
		s = Pop(s)
		foo = Peek(s)
	}

	b.StopTimer()
	stats := checkMem()
	b.Logf("memory usage:%d MB", stats.TotalAlloc/MiB)
	b.Logf("GC cycles: %d", stats.NumGC)
}

func checkMem() *runtime.MemStats {
	mem := &runtime.MemStats{}
	runtime.ReadMemStats(mem)
	return mem
}
