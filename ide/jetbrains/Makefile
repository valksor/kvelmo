# Mehrhof IntelliJ Plugin - Build Commands

# Java version - must match .github/workflows/ci.yml jetbrains job
JAVA_VERSION ?= 21
JAVA_HOME ?= $(shell /usr/libexec/java_home -v $(JAVA_VERSION) 2>/dev/null || echo "")

# Gradle wrapper
GRADLE = ./gradlew
ifeq ($(JAVA_HOME),)
    GRADLE_CMD = $(GRADLE)
else
    GRADLE_CMD = JAVA_HOME=$(JAVA_HOME) $(GRADLE)
endif

.PHONY: all build clean test coverage coverage-html run install help check verify publish deps wrapper stop

all: build ## Build the plugin (default)

build: ## Build the plugin ZIP
	$(GRADLE_CMD) buildPlugin

clean: ## Clean build artifacts
	$(GRADLE_CMD) clean

test: ## Run tests
	$(GRADLE_CMD) test

coverage: ## Run tests with coverage report
	$(GRADLE_CMD) test jacocoTestReport

coverage-html: coverage ## Open HTML coverage report
	open build/reports/jacoco/test/html/index.html

run: ## Run plugin in sandbox IDE for testing
	$(GRADLE_CMD) runIde

install: build ## Build and show install instructions
	@echo ""
	@echo "Plugin built successfully!"
	@echo ""
	@echo "To install:"
	@echo "  1. Open IntelliJ/GoLand"
	@echo "  2. Settings → Plugins → ⚙️ → Install Plugin from Disk"
	@echo "  3. Select: $(shell pwd)/build/distributions/mehrhof-intellij-*.zip"
	@echo "  4. Restart IDE"
	@echo ""

check: ## Compile and check for errors
	$(GRADLE_CMD) compileKotlin

verify: ## Verify plugin compatibility
	$(GRADLE_CMD) verifyPlugin

publish: ## Publish to JetBrains Marketplace (requires PUBLISH_TOKEN)
	$(GRADLE_CMD) publishPlugin

deps: ## Download and cache dependencies
	$(GRADLE_CMD) dependencies

wrapper: ## Update Gradle wrapper
	$(GRADLE_CMD) wrapper --gradle-version=8.10.2

stop: ## Stop Gradle/Kotlin daemons (use if build fails with Java version errors)
	$(GRADLE_CMD) --stop
	@pkill -f "kotlin-daemon" 2>/dev/null || true

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'
