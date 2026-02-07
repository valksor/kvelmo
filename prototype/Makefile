.PHONY: build build-embedder test quality install clean run hooks lefthook generate-licenses e2e e2e-fast e2e-check vscode-quality jetbrains-quality ide-quality quality-all sandbox-build sandbox-build-dev sandbox-run sandbox-interactive sandbox-push sandbox-ls sandbox-clean web-build web-dev web-install web-a11y web-test web-test-coverage

help: ## Outputs this help screen
	@grep -E '(^[a-zA-Z0-9_-]+:.*?##.*$$)|(^##)' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}{printf "\033[32m%-30s\033[0m %s\n", $$1, $$2}' | sed -e 's/\[32m##/[33m/'

# Build variables
BINARY_NAME := mehr
BUILD_DIR := ./build
CMD_DIR := ./cmd/mehr
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
BUILD_TIME := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS := -ldflags "-s -w -v -X github.com/valksor/go-toolkit/version.Version=$(VERSION) -X github.com/valksor/go-toolkit/version.Commit=$(COMMIT) -X github.com/valksor/go-toolkit/version.BuildTime=$(BUILD_TIME)"

# Default target
all: build ## Build the binary (default target)

build: generate-licenses web-build ## Compile the binary
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build -trimpath $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)
	@echo "Built $(BUILD_DIR)/$(BINARY_NAME)"

build-embedder: ## Compile the ONNX embedder sidecar (requires ONNX Runtime)
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 go build -trimpath $(LDFLAGS) -o $(BUILD_DIR)/mehr-embedder ./cmd/mehr-embedder
	@echo "Built $(BUILD_DIR)/mehr-embedder"

generate-licenses: ## Generate licenses.json from go.mod dependencies
	@echo "Generating dependency licenses..."
	@go run .github/gen_licenses.go

test: ## Run tests with coverage
	${MAKE} quality
	go test -v -cover ./...

race: ## Run race tests
	${MAKE} quality
	go test -v -race ./...

coverage: ## Run tests with race detection and coverage profile
	go test -race -covermode atomic -coverprofile=covprofile ./...

coverage-html: coverage ## Generate HTML coverage report
	@mkdir -p .coverage
	go tool cover -html=covprofile -o .coverage/coverage.html

quality: ## Run linter (golangci-lint)
	${MAKE} fmt
	golangci-lint run ./... --fix
	##govulncheck ./...
	${MAKE} check-alias

fmt: ## Format code with go fmt, goimports, and gofumpt
	go fmt ./...
	goimports -w .
	gofumpt -l -w .

install: build ## Install binary locally
	@INSTALL_DIR=""; \
	for dir in "$$HOME/.local/bin" "$$HOME/bin" "/usr/local/bin"; do \
		if [ -d "$$dir" ] && [ -w "$$dir" ]; then \
			INSTALL_DIR="$$dir"; \
			break; \
		fi; \
		if [ "$$dir" = "$$HOME/.local/bin" ] || [ "$$dir" = "$$HOME/bin" ]; then \
			if mkdir -p "$$dir" 2>/dev/null; then \
				INSTALL_DIR="$$dir"; \
				break; \
			fi; \
		fi; \
	done; \
	if [ -z "$$INSTALL_DIR" ]; then \
		INSTALL_DIR="/usr/local/bin"; \
	fi; \
	INSTALL_PATH="$$INSTALL_DIR/$(BINARY_NAME)"; \
	echo "Installing to $$INSTALL_PATH..."; \
	if [ -w "$$INSTALL_DIR" ]; then \
		cp $(BUILD_DIR)/$(BINARY_NAME) "$$INSTALL_PATH"; \
	else \
		echo "Requesting sudo access to install to $$INSTALL_DIR..."; \
		sudo cp $(BUILD_DIR)/$(BINARY_NAME) "$$INSTALL_PATH"; \
	fi; \
	echo "✓ Installed $(BINARY_NAME) to $$INSTALL_PATH"; \
	case ":$$PATH:" in \
		*":$$INSTALL_DIR:"*) ;; \
		*) \
			echo ""; \
			echo "⚠ $$INSTALL_DIR is not in your PATH"; \
			echo "Add it by running:"; \
			SHELL_NAME=$$(basename "$$SHELL"); \
			case "$$SHELL_NAME" in \
				bash) echo "  echo 'export PATH=\"$$INSTALL_DIR:\$$PATH\"' >> ~/.bashrc && source ~/.bashrc" ;; \
				zsh)  echo "  echo 'export PATH=\"$$INSTALL_DIR:\$$PATH\"' >> ~/.zshrc && source ~/.zshrc" ;; \
				*)    echo "  export PATH=\"$$INSTALL_DIR:\$$PATH\"" ;; \
			esac; \
			echo ""; \
		;; \
	esac; \
	if command -v $(BINARY_NAME) >/dev/null 2>&1; then \
		echo "Verifying installation..."; \
		$(BINARY_NAME) version; \
	fi

clean: ## Clean build artifacts
	rm -rf $(BUILD_DIR)
	rm -rf .coverage covprofile

run: build ## Run the binary (for development)
	$(BUILD_DIR)/$(BINARY_NAME)

run-args: build ## Run the binary with arguments (use ARGS=...)
	$(BUILD_DIR)/$(BINARY_NAME) $(ARGS)

tidy: clean ## Clean and tidy dependencies
	go mod tidy -e
	go get -d -v ./...

deps: ## Download dependencies
	go mod download

version: build ## Show version info
	$(BUILD_DIR)/$(BINARY_NAME) version

hooks: ## Configure git to use versioned hooks
	git config core.hooksPath .github/.githooks
	@echo "Git hooks configured to use .githooks/"

lefthook: ## Install and configure Lefthook pre-commit hooks
	go install github.com/evilmartians/lefthook@latest
	lefthook install
	@echo "Lefthook installed. Pre-commit hooks active."

check-alias:
	@alias_issues="$$(./.github/alias.sh || true)"; \
	if [ -n "$$alias_issues" ]; then \
		echo "❌ Unnecessary import alias detected:"; \
		echo "$$alias_issues"; \
		exit 1; \
	fi

# ──────────────────────────────────────────────────────────────────────────────
# IDE Plugin Quality
# ──────────────────────────────────────────────────────────────────────────────

## Run VS Code extension quality checks
vscode-quality:
	cd ide/vscode && make quality

## Run JetBrains plugin quality checks
jetbrains-quality:
	cd ide/jetbrains && make quality

## Run quality checks on all IDEs
ide-quality: vscode-quality jetbrains-quality

## Run ALL quality checks (Go + IDEs)
quality-all: quality ide-quality

# ──────────────────────────────────────────────────────────────────────────────
# React Frontend
# ──────────────────────────────────────────────────────────────────────────────

## Install React frontend dependencies
web-install:
	cd web && bun install

## Run local web accessibility checks (skipped in CI)
web-a11y:
	@if [ -z "$$CI" ]; then \
		echo "Running local web accessibility checks..."; \
		cd web && bun run lint:a11y; \
	else \
		echo "Skipping local web accessibility checks in CI"; \
	fi

## Build React frontend (outputs to internal/server/static/app/)
web-build: web-a11y
	cd web && bun run build

## Run React frontend dev server (with proxy to Go backend)
web-dev:
	cd web && bun run dev

## Run React frontend tests
web-test:
	cd web && bun run test:run

## Run React frontend tests with coverage
web-test-coverage:
	cd web && bun run test:coverage

# ──────────────────────────────────────────────────────────────────────────────
# E2E Tests (Local Manual Only)
# ──────────────────────────────────────────────────────────────────────────────
#
# Prerequisites:
#   - ZAI_API_KEY: ZAI API key for glm agent
#   - claude CLI installed and in PATH
#
# ──────────────────────────────────────────────────────────────────────────────

## Check E2E prerequisites
e2e-check:
	@echo "Checking E2E prerequisites..."
	@which claude >/dev/null || (echo "ERROR: claude CLI not found in PATH" && exit 1)
	@if test -n "$$ZAI_API_KEY"; then :; elif test -f .mehrhof/.env && grep -q "ZAI_API_KEY" .mehrhof/.env; then :; else echo "ERROR: ZAI_API_KEY not set (in environment or .mehrhof/.env)" && exit 1; fi
	@echo "✓ All prerequisites met!"

## Run fast E2E tests (~10 min, no git, no GitHub)
e2e-fast: build e2e-check
	@echo "Running fast E2E tests..."
	ZAI_API_KEY="$(ZAI_API_KEY)" \
	go test -v -tags=e2e_fast -timeout 20m ./e2e/fast/...

## Run E2E tests (alias for e2e-fast)
e2e: e2e-fast

# ──────────────────────────────────────────────────────────────────────────────
# Docker Sandbox
# ──────────────────────────────────────────────────────────────────────────────

SANDBOX_IMAGE := mehr-sandbox
SANDBOX_TAG := v1
SANDBOX_REGISTRY := valksor

## Build Docker Sandbox template with mehr pre-installed
sandbox-build:
	@echo "Building Docker Sandbox template..."
	docker build -f sandbox/Dockerfile.mehr -t $(SANDBOX_IMAGE):$(SANDBOX_TAG) .
	@echo "Built $(SANDBOX_IMAGE):$(SANDBOX_TAG)"

## Build sandbox from local source (for development)
## Note: Requires uncommenting local build section in Dockerfile.mehr
sandbox-build-dev: build
	@echo "Building Docker Sandbox template from local source..."
	@cp $(BUILD_DIR)/$(BINARY_NAME) sandbox/mehr-local
	docker build -f sandbox/Dockerfile.mehr \
		--build-arg MEHR_VERSION=local \
		-t $(SANDBOX_IMAGE):dev .
	@rm -f sandbox/mehr-local
	@echo "Built $(SANDBOX_IMAGE):dev"

## Run mehr in Docker Sandbox (current directory)
sandbox-run:
	@echo "Starting Docker Sandbox with mehr..."
	docker sandbox run --load-local-template -t $(SANDBOX_IMAGE):$(SANDBOX_TAG) $(PWD)

## Run sandbox in interactive mehr mode
sandbox-interactive:
	@echo "Starting Docker Sandbox with mehr interactive..."
	docker sandbox run --load-local-template -t $(SANDBOX_IMAGE):$(SANDBOX_TAG) $(PWD) -- mehr interactive

## Push sandbox template to Docker Hub
sandbox-push:
	docker tag $(SANDBOX_IMAGE):$(SANDBOX_TAG) $(SANDBOX_REGISTRY)/$(SANDBOX_IMAGE):$(SANDBOX_TAG)
	docker push $(SANDBOX_REGISTRY)/$(SANDBOX_IMAGE):$(SANDBOX_TAG)
	@echo "Pushed $(SANDBOX_REGISTRY)/$(SANDBOX_IMAGE):$(SANDBOX_TAG)"

## List running sandboxes
sandbox-ls:
	docker sandbox ls

## Clean up sandbox (removes the sandbox VM)
sandbox-clean:
	@echo "Removing sandbox..."
	-docker sandbox rm $(shell docker sandbox ls -q 2>/dev/null | head -1) 2>/dev/null || true
	@echo "Sandbox removed"
