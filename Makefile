.PHONY: build run test test-unit test-integration test-e2e test-bench test-coverage clean docker-build docker-run lint fmt

build:
	go build -o fake-batch-server cmd/server/main.go

run: build
	./fake-batch-server

test:
	go test -v ./...

test-unit:
	go test -v ./pkg/...

test-integration:
	go test -v ./pkg/handlers/...

test-e2e:
	go test -v ./test/... -run TestEndToEnd

test-bench:
	go test -bench=. -benchmem ./test/...

test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

clean:
	rm -f fake-batch-server coverage.out coverage.html

docker-build:
	docker build -t fake-batch-server .

docker-run: docker-build
	docker run -p 8080:8080 fake-batch-server

lint:
	golangci-lint run

fmt:
	go fmt ./...

