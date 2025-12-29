.PHONY: build test lint clean install pre-commit

# Build the binary
build:
	go build -buildvcs=false -trimpath -ldflags "-X main.version=dev" -o bin/rl .

# Run tests
test:
	go test -v ./...

# Run linters
lint:
	go fmt ./...
	go vet ./...

# Clean build artifacts
clean:
	rm -rf bin/

# Build and install locally
install:
	go install -buildvcs=false -trimpath -ldflags "-X main.version=dev" .

# Install pre-commit hook
pre-commit:
	@if [ ! -d .git ]; then \
		echo "Error: Not in a git repository"; \
		exit 1; \
	fi
	@mkdir -p .git/hooks
	@cp scripts/pre-commit .git/hooks/pre-commit
	@chmod +x .git/hooks/pre-commit
	@echo "Pre-commit hook installed successfully!"

