# Contributing

Contributions are welcome. For smaller changes feel free to send a PR directly, but for big stuff it probably makes sense to open an issue first to discuss it.

Please always try to clearly describe what problem you are trying to solve (intend) rather then just describing the change your are trying to make.

## Runbook

Below are various commands and procedures that might be useful for people trying to contribute to this package.

```
# run all tests
go test -v

# run all benchmarks
cp panicparse_test.go.disabled panicparse_test.go && go get -t . && go test -bench .

### FUZZING ###
# generate fuzz-corpus directory (takes > 10s, generated 434MiB of data)
FUZZ_CORPUS=true go test
# alternative to the above corpus generation (doesn't seem to work as well)
mkdir -p corpus && cp test-fixtures/*.txt corpus
# install/update go-fuzz
go get -u github.com/dvyukov/go-fuzz/go-fuzz github.com/dvyukov/go-fuzz/go-fuzz-build
# build the test program with necessary instrumentation
go-fuzz-build
# start fuzzing, then wait for a while to see if it finds any issues
go-fuzz
# cleanup
rm -rf gostackparse-fuzz.zip corpus

# refresh go.dev cache by visiting this URL with latest version
https://pkg.go.dev/github.com/DataDog/gostackparse@v0.4.0
```
