default: build vet

vet:
	go vet ./...

build:
	go get ./...

test: build
	go test -v ./...

static_build:
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o ./bin/drp
	ls -lh ./bin/

docker_build: static_build
	docker build -t dynport/drp .

docker_run: docker_build
	docker run --restart=always --name drp -d -p 80:8000 -p 8001:8001 dynport/drp
