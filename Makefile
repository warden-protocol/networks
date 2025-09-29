# Warden Networks - GenTx Validation Makefile
.PHONY: help install-dagger validate validate-batch validate-legacy validate-verbose test-tool clean

# Default target - now uses batch validation
validate: validate-batch ## Validate all mainnet GenTx files using batch processing (RECOMMENDED)

help: ## Show this help message
	@echo "Warden Networks GenTx Validation"
	@echo "================================="
	@echo ""
	@echo "🎯 RECOMMENDED: Use 'make validate' for batch validation"
	@echo ""
	@echo "Available commands:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "Requirements:"
	@echo "  - Dagger CLI (run 'make install-dagger' to install)"
	@echo ""

install-dagger: ## Install Dagger CLI (macOS/Linux)
	@echo "Installing Dagger CLI..."
	@if command -v brew >/dev/null 2>&1; then \
		echo "Using Homebrew..."; \
		brew install dagger/tap/dagger; \
	else \
		echo "Using direct install script..."; \
		curl -fsSL https://dl.dagger.io/dagger/install.sh | BIN_DIR=$$HOME/.local/bin sh; \
		echo "Make sure $$HOME/.local/bin is in your PATH"; \
	fi
	@echo "Verifying installation..."
	@dagger version

validate-batch: ## Validate all GenTx files together (RECOMMENDED - matches production genesis process)
	@echo "🚀 Running batch GenTx validation (all files processed together)..."
	@cd ci && dagger call validate-all-gentx-together --source=.. --network=mainnet

validate-legacy: ## Validate GenTx files individually (old method)
	@echo "🚀 Running legacy GenTx validation (files processed individually)..."
	@cd ci && dagger call validate-gentx --source=.. --network=mainnet

validate-verbose: ## Run batch validation with detailed output
	@echo "🚀 Running batch GenTx validation with verbose output..."
	@cd ci && dagger call --progress=plain validate-all-gentx-together --source=.. --network=mainnet

test-tool: ## Test that check-genesis tool builds correctly  
	@echo "🔧 Testing check-genesis tool build..."
	@cd ci && dagger call test-check-genesis-tool --source=..

validate-custom: ## Validate with custom parameters (use NETWORK, WARDEND_VERSION, GO_VERSION env vars)
	@echo "🚀 Running batch GenTx validation with custom parameters..."
	@cd ci && dagger call validate-all-gentx-together \
		--source=.. \
		--network=$(or $(NETWORK),mainnet) \
		--wardend-version=$(or $(WARDEND_VERSION),v0.7.0) \
		--go-version=$(or $(GO_VERSION),1.24)

# Development targets
dev-validate: validate-verbose ## Alias for validate-verbose (common during development)

dev-test: test-tool validate-verbose ## Run both tool test and validation with verbose output

clean: ## Clean Dagger cache and temporary files
	@echo "🧹 Cleaning Dagger cache..."
	@dagger query -f - <<< '{ core { cacheEntries { clear } } }'
	@echo "Cache cleaned!"

# Quick validation for different scenarios
validate-pr: ## Simulate PR validation (validates changed files only)
	@echo "📋 This would validate only changed GenTx files in a real PR"
	@echo "For now, running full validation..."
	@$(MAKE) validate-verbose

# Help with debugging
debug-info: ## Show debug information about the environment
	@echo "🔍 Debug Information"
	@echo "==================="
	@echo "Dagger version: $$(dagger version 2>/dev/null || echo 'Not installed')"
	@echo "Docker status: $$(docker version --format '{{.Server.Version}}' 2>/dev/null || echo 'Not available')"
	@echo "Current directory: $$(pwd)"
	@echo "GenTx files found:"
	@find mainnet/gentx -name "*.json" 2>/dev/null | head -5 | sed 's/^/  /' || echo "  No mainnet/gentx directory found"
	@echo ""
	@echo "To install Dagger: make install-dagger"
	@echo "To validate: make validate"

# Example usage with different parameters
examples: ## Show example commands
	@echo "📖 Example Commands"
	@echo "=================="
	@echo ""
	@echo "🎯 RECOMMENDED - Batch validation (all files processed together):"
	@echo "  make validate           # Default batch validation"
	@echo "  make validate-batch     # Same as above (explicit)"
	@echo "  make validate-verbose   # Batch validation with detailed output"
	@echo ""
	@echo "Legacy validation (files processed individually):"
	@echo "  make validate-legacy    # Old method for comparison"
	@echo ""
	@echo "Testing and debugging:"
	@echo "  make test-tool          # Test tool build"
	@echo "  make debug-info         # Environment information"
	@echo ""
	@echo "Custom parameters:"
	@echo "  WARDEND_VERSION=v0.6.0 make validate-custom"
	@echo "  GO_VERSION=1.21 NETWORK=mainnet make validate-custom"
	@echo ""
	@echo "Development workflow:"
	@echo "  make dev-test          # Test tool + batch validate with output"
	@echo ""
	@echo "🔍 Why batch validation?"
	@echo "  • Matches production genesis collection process"
	@echo "  • All gentx files processed together (more accurate)"
	@echo "  • Better performance (no environment rebuilding per file)"
	@echo "  • Detects inter-file conflicts (duplicate validators, etc.)"
	@echo ""
