# ember2go Makefile - Advanced ARM64 optimizations for Apple Silicon

# Build variables
BINARY_NAME=ember2go
VERSION=1.0.0
BUILD_TIME=$(shell date +%Y-%m-%dT%H:%M:%S%z)
GIT_COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Go build flags for maximum ARM64 optimization
GOARM64=v8.0,lse,crypto
GOMAXPROCS=$(shell sysctl -n hw.ncpu)
CGO_ENABLED=0

# Optimization flags
LDFLAGS=-w -s \
	-X main.version=$(VERSION) \
	-X main.buildTime=$(BUILD_TIME) \
	-X main.gitCommit=$(GIT_COMMIT)

GCFLAGS=-l=4 -B -C

# PGO profile location
PGO_PROFILE=default.pgo

.PHONY: all clean build build-optimized build-universal install-deps profile-build benchmark test

# Default build (ARM64 optimized)
all: build-optimized

# Standard build
build:
	GOOS=darwin GOARCH=arm64 GOARM64=$(GOARM64) CGO_ENABLED=$(CGO_ENABLED) \
	go build -o $(BINARY_NAME) ./cmd/$(BINARY_NAME)

# Highly optimized build for ARM64
build-optimized:
	@echo "Building optimized ember2go for Apple Silicon..."
	GOOS=darwin GOARCH=arm64 GOARM64=$(GOARM64) CGO_ENABLED=$(CGO_ENABLED) GOMAXPROCS=$(GOMAXPROCS) \
	go build -ldflags="$(LDFLAGS)" -gcflags="$(GCFLAGS)" -o $(BINARY_NAME) ./cmd/$(BINARY_NAME)
	@echo "✅ Optimized build complete: $(shell ls -lh $(BINARY_NAME) | awk '{print $$5}') binary"

# Profile-guided optimization build (requires default.pgo)
build-pgo:
	@if [ -f $(PGO_PROFILE) ]; then \
		echo "Building with Profile-Guided Optimization..."; \
		GOOS=darwin GOARCH=arm64 GOARM64=$(GOARM64) CGO_ENABLED=$(CGO_ENABLED) GOMAXPROCS=$(GOMAXPROCS) \
		go build -pgo=$(PGO_PROFILE) -ldflags="$(LDFLAGS)" -gcflags="$(GCFLAGS)" -o $(BINARY_NAME) ./cmd/$(BINARY_NAME); \
		echo "✅ PGO build complete: $(shell ls -lh $(BINARY_NAME) | awk '{print $$5}') binary"; \
	else \
		echo "⚠️  No $(PGO_PROFILE) found. Run 'make profile' first or use 'make build-optimized'"; \
		$(MAKE) build-optimized; \
	fi

# Universal binary (Intel + ARM64)
build-universal:
	@echo "Building universal binary..."
	@mkdir -p build
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=$(CGO_ENABLED) \
	go build -ldflags="$(LDFLAGS)" -gcflags="$(GCFLAGS)" -o build/$(BINARY_NAME)-amd64 ./cmd/$(BINARY_NAME)
	GOOS=darwin GOARCH=arm64 GOARM64=$(GOARM64) CGO_ENABLED=$(CGO_ENABLED) \
	go build -ldflags="$(LDFLAGS)" -gcflags="$(GCFLAGS)" -o build/$(BINARY_NAME)-arm64 ./cmd/$(BINARY_NAME)
	lipo -create -output $(BINARY_NAME) build/$(BINARY_NAME)-amd64 build/$(BINARY_NAME)-arm64
	@echo "✅ Universal binary created: $(shell ls -lh $(BINARY_NAME) | awk '{print $$5}')"
	@echo "Architectures: $(shell lipo -info $(BINARY_NAME))"

# Cross-platform builds
build-all:
	@echo "Building for all platforms..."
	@mkdir -p dist
	GOOS=darwin GOARCH=arm64 GOARM64=$(GOARM64) CGO_ENABLED=$(CGO_ENABLED) \
	go build -ldflags="$(LDFLAGS)" -o dist/$(BINARY_NAME)-darwin-arm64 ./cmd/$(BINARY_NAME)
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=$(CGO_ENABLED) \
	go build -ldflags="$(LDFLAGS)" -o dist/$(BINARY_NAME)-darwin-amd64 ./cmd/$(BINARY_NAME)
	GOOS=linux GOARCH=arm64 CGO_ENABLED=$(CGO_ENABLED) \
	go build -ldflags="$(LDFLAGS)" -o dist/$(BINARY_NAME)-linux-arm64 ./cmd/$(BINARY_NAME)
	GOOS=linux GOARCH=amd64 CGO_ENABLED=$(CGO_ENABLED) \
	go build -ldflags="$(LDFLAGS)" -o dist/$(BINARY_NAME)-linux-amd64 ./cmd/$(BINARY_NAME)
	@echo "✅ Cross-platform builds complete:"
	@ls -lh dist/

# Generate PGO profile
profile:
	@echo "Generating PGO profile..."
	@echo "⚠️  This requires running ember2go with typical workloads"
	@echo "1. Build a basic version first"
	$(MAKE) build
	@echo "2. Run with CPU profiling:"
	@echo "   CPUPROFILE=cpu.prof ./$(BINARY_NAME) list-disks"
	@echo "   CPUPROFILE=cpu.prof ./$(BINARY_NAME) list-editions /path/to/test.iso"
	@echo "3. Convert profile:"
	@echo "   go tool pprof -proto cpu.prof > $(PGO_PROFILE)"
	@echo "4. Then run: make build-pgo"

# Benchmark performance
benchmark:
	@echo "Running performance benchmarks..."
	@echo "Testing disk operations..."
	@/usr/bin/time ./$(BINARY_NAME) list-disks >/dev/null
	@echo "\nBinary size: $(shell ls -lh $(BINARY_NAME) | awk '{print $$5}')"
	@echo "Architectures: $(shell lipo -info $(BINARY_NAME) 2>/dev/null || echo 'Single architecture')"

# Install dependencies optimized for ARM64
install-deps:
	@echo "Installing dependencies optimized for Apple Silicon..."
	@if command -v brew >/dev/null 2>&1; then \
		echo "Installing wimlib..."; \
		brew install wimlib; \
		echo "Installing optional tools..."; \
		brew install hivex || echo "hivex installation failed (optional)"; \
	else \
		echo "❌ Homebrew not found. Please install Homebrew first."; \
		exit 1; \
	fi

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -f $(BINARY_NAME)
	rm -rf build/
	rm -rf dist/
	rm -f *.prof
	rm -f $(PGO_PROFILE)

# Install to system
install: build-optimized
	@echo "Installing $(BINARY_NAME) to /usr/local/bin..."
	sudo cp $(BINARY_NAME) /usr/local/bin/
	sudo chmod +x /usr/local/bin/$(BINARY_NAME)
	@echo "✅ Installed. Run 'ember2go --help' to verify."

# Development helpers
dev-setup:
	@echo "Setting up development environment..."
	go mod tidy
	go mod download
	$(MAKE) install-deps

# Show build information
info:
	@echo "Build Information:"
	@echo "  Go version: $(shell go version)"
	@echo "  Target arch: darwin/arm64"
	@echo "  GOARM64: $(GOARM64)"
	@echo "  Max procs: $(GOMAXPROCS)"
	@echo "  CGO: $(CGO_ENABLED)"
	@echo "  Git commit: $(GIT_COMMIT)"
	@echo ""
	@echo "System Information:"
	@echo "  CPU cores: $(shell sysctl -n hw.ncpu)"
	@echo "  CPU brand: $(shell sysctl -n machdep.cpu.brand_string)"
	@echo "  Memory: $(shell echo $$(($$(sysctl -n hw.memsize) / 1024 / 1024 / 1024)))GB"

help:
	@echo "ember2go Makefile - Advanced ARM64 optimizations"
	@echo ""
	@echo "Targets:"
	@echo "  build-optimized  Build highly optimized ARM64 binary (default)"
	@echo "  build-pgo        Build with Profile-Guided Optimization"
	@echo "  build-universal  Build universal binary (Intel + ARM64)"
	@echo "  build-all        Build for all platforms"
	@echo "  profile          Generate PGO profile"
	@echo "  benchmark        Run performance benchmarks"
	@echo "  install-deps     Install dependencies"
	@echo "  test             Run tests"
	@echo "  install          Install to system"
	@echo "  clean            Clean build artifacts"
	@echo "  info             Show build information"
	@echo ""
	@echo "Examples:"
	@echo "  make build-optimized    # Best performance for Apple Silicon"
	@echo "  make build-universal    # Compatible with Intel + ARM Macs"
	@echo "  make profile && make build-pgo  # Maximum optimization with PGO"