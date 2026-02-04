VERSION ?= dev
BINARY  := hls2rtsp

.PHONY: build test lint docker clean

build:
	go build -ldflags "-s -w -X main.version=$(VERSION)" -o $(BINARY) ./cmd/hls2rtsp

test:
	go test -v -race ./...

lint:
	golangci-lint run ./...

docker:
	docker build --build-arg VERSION=$(VERSION) -t $(BINARY):$(VERSION) .

clean:
	rm -f $(BINARY)
