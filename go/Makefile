GOOS=linux
GOARCH=amd64

NAME=meetup

clean:
	@echo ">> cleaning..."
	@rm -f $(NAME)

build: clean
	@echo ">> building..."
	@ go build -ldflags "-X main.Version=$(VERSION) -X main.CommitID=$(COMMIT_ID)" -o $(NAME) ./parser.go
	@chmod +x $(NAME)
