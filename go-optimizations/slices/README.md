go build -gcflags=-m=4 slice_index.go 2> index.txt
go build -gcflags=-m=4 slice_pointer.go 2> pointer.txt

diff index.txt pointer.txt

index:
> DOT tc(1) int # slice_index.go:15 main.counter int \
> .  INDEX tc(1) main.DTO # slice_index.go:15 main.DTO

pointer:
> DOTPTR tc(1) int # slice_pointer.go:15 main.counter int \
> .   ADDR tc(1) PTR-*DTO # slice_pointer.go:15 PTR-*DTO \
> .   .   INDEX tc(1) main.DTO # slice_pointer.go:15 main.DTO
