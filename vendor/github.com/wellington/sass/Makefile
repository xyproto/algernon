test:
	go list -f '{{if len .TestGoFiles}}"go test -short {{.ImportPath}}"{{end}}' ./... | grep -v /vendor/ | xargs -L 1 sh -c

cover:
	go list -f '{{if len .TestGoFiles}}"go test -cover -short {{.ImportPath}}"{{end}}' ./... | grep -v /vendor/ | xargs -L 1 sh -c
race:
	go list -f '{{if len .TestGoFiles}}"go test -race -short {{.ImportPath}}"{{end}}' ./... | grep -v /vendor/ | xargs -L 1 sh -c
