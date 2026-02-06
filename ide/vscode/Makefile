.PHONY: all build watch test lint lint-fix format format-check quality package install clean deps

all: build

# Install dependencies
deps:
	bun install

# Build the extension
build: deps
	bun run compile

# Watch for changes
watch: deps
	bun run watch

# Run tests (bun test, no VS Code needed)
test: deps
	bun run test

# Run tests with coverage
test-coverage: deps
	bun run test:coverage

# Run integration tests (requires VS Code)
test-integration: deps
	bun run compile && bun run compile:test
	bun run test:integration

# Run linter
lint: deps
	bun run lint

# Run linter with auto-fix
lint-fix: deps
	bun run lint:fix

# Format code with Prettier
format: deps
	bun run format

# Check formatting (for CI)
format-check: deps
	bun run format:check

# Run all quality checks (lint + format)
quality: deps
	bun run quality

# Package the extension
package: build
	bun run package

# Install the extension in VS Code
install: package
	code --install-extension mehrhof-*.vsix

# Clean build artifacts
clean:
	rm -rf dist out node_modules *.vsix

# Help
help:
	@echo "Available targets:"
	@echo "  deps         - Install bun dependencies"
	@echo "  build        - Build the extension"
	@echo "  watch        - Watch for changes and rebuild"
	@echo "  test         - Run unit tests (fast, no VS Code)"
	@echo "  test-coverage - Run unit tests with coverage"
	@echo "  test-integration - Run integration tests (requires VS Code)"
	@echo "  lint         - Run ESLint"
	@echo "  lint-fix     - Run ESLint with auto-fix"
	@echo "  format       - Format code with Prettier"
	@echo "  format-check - Check formatting (for CI)"
	@echo "  quality      - Run all quality checks (lint + format)"
	@echo "  package      - Create .vsix package"
	@echo "  install      - Install extension in VS Code"
	@echo "  clean        - Remove build artifacts"
