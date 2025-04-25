.PHONY: build test fmt clean

build:
	go build ./...

test:
	go test ./...

fmt:
	go fmt ./...

clean:
	rm -f connman-trigger
