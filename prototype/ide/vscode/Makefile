.PHONY: all build watch test lint lint-fix package install clean deps

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

# Compile tests
compile-test: deps
	bun run compile:test

# Run tests (requires VS Code)
test: deps compile-test
	bun run test

# Run unit tests only (fast, no VS Code)
test-unit: deps
	bun run test:unit

# Run linter
lint: deps
	bun run lint

# Run linter with auto-fix
lint-fix: deps
	bun run lint:fix

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
	@echo "  compile-test - Compile test files"
	@echo "  test         - Run integration tests (requires VS Code)"
	@echo "  test-unit    - Run unit tests (fast, no VS Code)"
	@echo "  lint         - Run ESLint"
	@echo "  lint-fix     - Run ESLint with auto-fix"
	@echo "  package      - Create .vsix package"
	@echo "  install      - Install extension in VS Code"
	@echo "  clean        - Remove build artifacts"
