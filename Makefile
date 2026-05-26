VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X main.version=$(VERSION)"
BINARY  := eml-to-jsonl

# macOS Developer ID signing / notarization (see nlink-jp/.github
# CONVENTIONS.md §Code Signing). Defaults match any Developer ID
# Application cert in the keychain and the org-standard notary
# profile. Builds without these fall back to ad-hoc / un-notarized
# with a one-line warning — see scripts/codesign-darwin.sh.
CODESIGN_IDENTITY ?= Developer ID Application
NOTARY_PROFILE    ?= nlink-jp-notary

.PHONY: build test vet lint check clean setup build-all package

build:
	@mkdir -p dist
	go build $(LDFLAGS) -o dist/$(BINARY) .
	@scripts/codesign-darwin.sh dist/$(BINARY) "$(CODESIGN_IDENTITY)"

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
	rm -rf dist/

build-all:
	mkdir -p dist
	GOOS=linux   GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY)-linux-amd64   . ; \
	GOOS=linux   GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY)-linux-arm64   . ; \
	GOOS=darwin  GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY)-darwin-amd64  . ; \
	GOOS=darwin  GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY)-darwin-arm64  . ; \
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY)-windows-amd64.exe .
	@scripts/codesign-darwin.sh dist/$(BINARY)-darwin-amd64 "$(CODESIGN_IDENTITY)"
	@scripts/codesign-darwin.sh dist/$(BINARY)-darwin-arm64 "$(CODESIGN_IDENTITY)"
	@echo "Cross-compiled binaries in dist/"

## package: Build all platforms, zip with versioned naming + README, notarize darwin → dist/
package: build-all
	@cd dist && for f in $(BINARY)-*; do \
		case "$$f" in *.zip) continue ;; esac; \
		suffix=$${f#$(BINARY)-}; \
		suffix=$${suffix%%.exe}; \
		ext=""; case "$$f" in *.exe) ext=".exe" ;; esac; \
		cp ../README.md .; \
		stage="$$(dirname "$$f")/_pkg"; rm -rf "$$stage"; mkdir -p "$$stage"; \
		cp "$$f" "$$stage/$(BINARY)$$ext"; \
		zip -j "$(BINARY)-$(VERSION)-$${suffix}.zip" "$$stage/$(BINARY)$$ext" README.md; \
		rm -rf "$$stage"; \
		rm -f README.md; \
	done
	@scripts/notarize-darwin.sh dist/$(BINARY)-$(VERSION)-darwin-amd64.zip "$(NOTARY_PROFILE)"
	@scripts/notarize-darwin.sh dist/$(BINARY)-$(VERSION)-darwin-arm64.zip "$(NOTARY_PROFILE)"
