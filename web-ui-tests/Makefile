.PHONY: all deps test test-smoke test-ui test-debug lint lint-fix format format-check quality install-browsers clean help

all: test

# Install dependencies
deps:
	bun install

# Run all tests
test: deps
	bun run test

# Run smoke tests only
test-smoke: deps
	bun run test:smoke

# Run tests with UI
test-ui: deps
	bun run test:ui

# Run tests in debug mode
test-debug: deps
	bun run test:debug

# Install Playwright browsers
install-browsers: deps
	bun run install:browsers

# Run ESLint
lint: deps
	bun run lint

# Run ESLint with auto-fix
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

# Show HTML test report
show-report:
	bun run show-report

# Clean build artifacts
clean:
	rm -rf node_modules test-results playwright-report

# Help
help:
	@echo "Available targets:"
	@echo "  deps            - Install bun dependencies"
	@echo "  test            - Run all tests"
	@echo "  test-smoke      - Run smoke tests only"
	@echo "  test-ui         - Run tests with UI"
	@echo "  test-debug      - Run tests in debug mode"
	@echo "  install-browsers - Install Playwright browsers"
	@echo "  lint            - Run ESLint"
	@echo "  lint-fix        - Run ESLint with auto-fix"
	@echo "  format          - Format code with Prettier"
	@echo "  format-check    - Check formatting (for CI)"
	@echo "  quality         - Run all quality checks (lint + format)"
	@echo "  show-report     - Show HTML test report"
	@echo "  clean           - Remove build artifacts"
