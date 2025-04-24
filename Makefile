MAKEFLAGS=--no-builtin-rules --no-builtin-variables --always-make

fmt:
	go tool golangci-lint fmt

lint:
	go tool golangci-lint run

test:
	go test -v ./...
