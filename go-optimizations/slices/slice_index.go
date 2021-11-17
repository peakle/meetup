package main

type DTO struct {
	counter int
}

//go:noinline
func main() {
	var (
		dtos []DTO
		sum  int
	)

	for i := range dtos {
		sum += dtos[i].counter
	}
}
