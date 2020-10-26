# Run tests

To run tests simply run

```sh
go test ./... -coverprofile=coverage.out
```

To deep dive into test coverage, run the following command to see the result in your terminal

```sh
go tool cover -func=coverage.out
```

or the following to see the result in your browser

```sh
go tool cover -html=coverage.out
```
