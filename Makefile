.PHONY: build run test clean docker-build docker-run

build:
	go build -o fake-batch-server cmd/server/main.go

run: build
	./fake-batch-server

test:
	go test -v ./...

clean:
	rm -f fake-batch-server

docker-build:
	docker build -t fake-batch-server .

docker-run: docker-build
	docker run -p 8080:8080 fake-batch-server

lint:
	golangci-lint run

fmt:
	go fmt ./...
