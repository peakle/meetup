package go_optimizations

import (
	"database/sql"
	"testing"
)

type RowHeap struct {
	Str *string `db:"str"`
}

type RowStack struct {
	Str sql.NullString `db:"str"`
}

// global var disable compiler loop optimization
var (
	globalRowHeap  RowHeap
	globalRowStack RowStack
)

func BenchmarkCreateRowHeap(b *testing.B) {
	for i := 0; i < b.N; i++ {
		globalRowHeap = CreateRowHeap()
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
