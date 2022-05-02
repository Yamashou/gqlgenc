MAKEFLAGS=--no-builtin-rules --no-builtin-variables --always-make

fmt:
	gofumpt -l -w .

lint:
	golangci-lint cache clean && golangci-lint run

test:
	go test -v ./...
