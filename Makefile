# Every target here is a task, not a file. Without this, `build` collides with
# the build/ directory and `make build` becomes a no-op after the first run --
# so every contributor after their first build silently tests a stale binary.
.PHONY: build build-all build-glibc build-musl build-musl-quick \
	test check fmt fmt-check vet lint vulncheck sbom sbom-check license-check \
	test-coverage test-coverage-clean clean docker-build docker-shell docker-stop

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
BINARY_NAME=fpm-exporter
VERSION?=dev
COMMIT?=$(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE?=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS=-w -s -X 'main.version=$(VERSION)' -X 'main.commit=$(COMMIT)' -X 'main.date=$(DATE)' 
BUILD_DIR=build
DOCKER_REPO=cboxdk
IMAGE_NAME=php:8.4-fpm-bookworm
CONTAINER_NAME=cbox-dev

# Build all platforms (works on both glibc and musl systems)
build-all:
	mkdir -p $(BUILD_DIR)
	@echo "Building static binaries for all platforms..."
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/fpm-exporter-linux-amd64 .
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(GOBUILD) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/fpm-exporter-linux-arm64 .
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GOBUILD) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/fpm-exporter-darwin-amd64 .
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 $(GOBUILD) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/fpm-exporter-darwin-arm64 .

# Quick local build (current platform)
build:
	mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 $(GOBUILD) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME) .

# Legacy aliases for backwards compatibility
build-glibc: build-all
build-musl: build-all
build-musl-quick: build

test:
	$(GOTEST) -race -cover ./...

# The same gate CI runs.
check: fmt-check vet lint test vulncheck sbom-check license-check

fmt:
	gofmt -w .

fmt-check:
	@unformatted=$$(gofmt -l .); \
	if [ -n "$$unformatted" ]; then \
		echo "These files are not gofmt-clean:"; \
		echo "$$unformatted"; \
		exit 1; \
	fi

vet:
	$(GOCMD) vet ./...

lint:
	golangci-lint run

vulncheck:
	$(GOCMD) run golang.org/x/vuln/cmd/govulncheck@latest ./...

# Deterministic CycloneDX 1.5 SBOM: no serial number, no timestamp, so it only
# changes when the dependencies do.
sbom:
	$(GOCMD) run github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod@v1.10.0 \
		mod -json -licenses -assert-licenses -noserial -notimestamp \
		-output-version 1.5 -output sbom.json .
	$(GOCMD) run ./tools/sbomnorm sbom.json

sbom-check: sbom
	git diff --exit-code sbom.json

license-check:
	$(GOCMD) run ./tools/licensecheck

test-coverage:
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -func=coverage.out
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report saved to coverage.html"

test-coverage-clean:
	rm -f coverage.out coverage.html

docker-build:
	docker build -t $(IMAGE_NAME) .

docker-shell:
	docker run -it --rm \
		--name $(CONTAINER_NAME) \
		-v $(CURDIR):/app \
		-w /app \
		$(IMAGE_NAME) bash

shell:
	docker exec -it $(CONTAINER_NAME) bash

docker-stop:
	docker stop $(CONTAINER_NAME) || true

clean:
	rm -rf $(BUILD_DIR) dist coverage.out coverage.html
