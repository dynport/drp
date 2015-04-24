default: build vet

vet:
	go vet ./...

build:
	go get ./...

test:
	go test -v ./...
