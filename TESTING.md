# Run tests

To run tests simply run

```shell script
go test ./... -coverprofile=coverage.out
```

To deep dive into test coverage, run the following command to see the result in your terminal

```shell script
go tool cover -func=coverage.out
```

or the following to see the result in your browser

```shell script
go tool cover -html=coverage.out
```


# End-to-End Testing

The `generator` package contains tests which 
run the full generator pipeline
and support assertions on the generated code.

To create such a test, 
copy one of the directories in `generator/testdata`
and modify the files to express your test case,
then run the tests with `go test ./generator/testdata/...`.
The test will check that the generated code compiles
and that the generated code matches the files in the
`expected` directory.
