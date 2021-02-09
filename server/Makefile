GOOS=linux
GOARCH=amd64

NAME=meetup_server

clean:
	@echo ">> cleaning..."
	@rm -f $(NAME)

build: clean
	@echo ">> building..."
	@ CGO_ENABLED=0 GOOS=${GOOS} GOARCH=${GOARCH} \
	    go build -ldflags "-X main.Version=$(VERSION) -X main.CommitID=$(COMMIT_ID)" -o ../$(NAME) ./server.go
	@chmod +x ../$(NAME)
