.PHONY: build run test fmt vet clean

build:
	go build ./...

run:
	go run .

test:
	go test ./...

fmt:
	gofmt -w .

vet:
	go vet ./...

clean:
	rm -f 55h
