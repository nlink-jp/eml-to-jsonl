VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X main.version=$(VERSION)"
BINARY  := lite-eml

.PHONY: build test vet lint check clean setup build-all

build:
	go build $(LDFLAGS) -o bin/$(BINARY) .

test:
	go test ./...

vet:
	go vet ./...

lint:
	golangci-lint run ./...

check: vet lint test build
	govulncheck ./...

setup:
	mkdir -p .git/hooks
	cp scripts/hooks/pre-commit .git/hooks/pre-commit
	cp scripts/hooks/pre-push   .git/hooks/pre-push
	chmod +x .git/hooks/pre-commit .git/hooks/pre-push

clean:
	rm -rf bin/ dist/

build-all:
	mkdir -p dist
	GOOS=linux   GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY)-linux-amd64   . ; \
	GOOS=linux   GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY)-linux-arm64   . ; \
	GOOS=darwin  GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY)-darwin-amd64  . ; \
	GOOS=darwin  GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY)-darwin-arm64  . ; \
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY)-windows-amd64.exe .
	@echo "Cross-compiled binaries in dist/"
