default: build vet

vet:
	go vet ./...

build:
	go get ./...

test:
	go test -v ./...

docker_build:
	docker build -t dynport/drp .

docker_run: docker_build
	docker run --restart=always --name drp -d -p 80:8000 -p 8001:8001 dynport/drp
