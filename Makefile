test: fmt vet
	go test ./... -coverprofile cover.out

fmt:
	go fmt ./...

vet:
	go vet ./...

build: fmt vet test
	go build -a -o bin/go-naivecoin *.go