MAKEFLAGS=--no-builtin-rules --no-builtin-variables --always-make

fmt:
	goimports -local github.com/Yamashou/gqlgenc -w . && gofumpt -extra -w . && gci write -s Standard -s Default .

lint:
	golangci-lint cache clean && golangci-lint run

test:
	go test -v ./...
