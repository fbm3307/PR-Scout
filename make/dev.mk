# =============================================================================
# Environment Check
# =============================================================================

.PHONY: doctor
doctor: ## Check if dev prerequisites are installed
	@echo -e "$(YELLOW)Checking development prerequisites...$(NC)"
	@ok=true; \
	check_cmd() { \
		if command -v "$$1" >/dev/null 2>&1; then \
			ver=$$($$2 2>&1 | head -1); \
			echo -e "  $(GREEN)✓$(NC) $$1  $$ver"; \
		else \
			echo -e "  $(RED)✗$(NC) $$1  -- not found ($$3)"; \
			ok=false; \
		fi; \
	}; \
	check_cmd go    "go version"    "https://go.dev/dl/"; \
	check_cmd node  "node --version" "https://nodejs.org/"; \
	check_cmd npm   "npm --version"  "https://nodejs.org/"; \
	echo ""; \
	echo -e "$(YELLOW)Configuration files:$(NC)"; \
	check_file() { \
		if [ -f "$$1" ]; then \
			echo -e "  $(GREEN)✓$(NC) $$1"; \
		else \
			echo -e "  $(RED)✗$(NC) $$1  -- not found ($$2)"; \
			ok=false; \
		fi; \
	}; \
	check_file deploy/config/pr-scout.yaml "create deploy/config/pr-scout.yaml — see README"; \
	check_file deploy/config/.env          "create deploy/config/.env — see README"; \
	echo ""; \
	if $$ok; then \
		echo -e "$(GREEN)✅ All checks passed$(NC)"; \
	else \
		echo -e "$(RED)❌ Some checks failed -- see above$(NC)"; \
		exit 1; \
	fi

# =============================================================================
# Development Workflow
# =============================================================================

.PHONY: build
build: ## Build Go binary
	@echo -e "$(YELLOW)Building pr-scout...$(NC)"
	@go build -o bin/pr-scout ./cmd/pr-scout
	@echo -e "$(GREEN)✅ Build complete: bin/pr-scout$(NC)"

.PHONY: dev
dev: build ## Start backend + dashboard dev server
	@-pkill -f 'bin/pr-scout' 2>/dev/null; true
	@-pkill -f 'web/dashboard.*vite' 2>/dev/null; true
	@sleep 0.3
	@echo -e "$(GREEN)Starting development environment...$(NC)"
	@echo -e "$(BLUE)  Go backend:  localhost:8080$(NC)"
	@echo -e "$(BLUE)  Dashboard:   localhost:5173$(NC)"
	@echo ""
	@trap 'kill 0' EXIT; \
		./bin/pr-scout & SCOUT_PID=$$!; \
		sleep 1; \
		if ! kill -0 $$SCOUT_PID 2>/dev/null; then \
			echo -e "\n$(RED)ERROR: pr-scout backend failed to start$(NC)" >&2; \
			exit 1; \
		fi; \
		echo -e "$(GREEN)✅ Backend running (pid $$SCOUT_PID)$(NC)"; \
		cd web/dashboard && npm run dev

.PHONY: dev-stop
dev-stop: ## Stop all dev services
	@echo -e "$(YELLOW)Stopping development services...$(NC)"
	@-pkill -f 'bin/pr-scout' 2>/dev/null; true
	@-pkill -f 'web/dashboard.*vite' 2>/dev/null; true
	@echo -e "$(GREEN)✅ All services stopped$(NC)"

.PHONY: scan
scan: ## Trigger a PR scan
	@echo -e "$(YELLOW)Triggering scan...$(NC)"
	@curl -s -X POST http://localhost:8080/api/v1/scan | python3 -m json.tool 2>/dev/null || \
		curl -s -X POST http://localhost:8080/api/v1/scan
	@echo ""

# =============================================================================
# Testing & Linting
# =============================================================================

.PHONY: test
test: test-go test-dashboard ## Run all tests

.PHONY: test-go
test-go: ## Run Go tests
	@echo -e "$(YELLOW)Running Go tests...$(NC)"
	@go test -v -race ./pkg/... ./cmd/...
	@echo -e "$(GREEN)✅ Go tests passed$(NC)"

.PHONY: test-dashboard
test-dashboard: ## Run dashboard tests
	@echo -e "$(YELLOW)Running dashboard tests...$(NC)"
	@cd web/dashboard && npm run test:run
	@echo -e "$(GREEN)✅ Dashboard tests passed$(NC)"

.PHONY: lint
lint: ## Run linters
	@echo -e "$(YELLOW)Running Go linter...$(NC)"
	@golangci-lint run --timeout=5m 2>/dev/null || go vet ./...
	@echo -e "$(YELLOW)Running dashboard linter...$(NC)"
	@cd web/dashboard && npm run lint 2>/dev/null || true
	@echo -e "$(GREEN)✅ Lint complete$(NC)"

.PHONY: fmt
fmt: ## Format code
	@go fmt ./...
	@echo -e "$(GREEN)✅ Code formatted$(NC)"

# =============================================================================
# Dependencies
# =============================================================================

.PHONY: setup
setup: ## Install all dependencies and bootstrap config
	@echo -e "$(YELLOW)Installing Go dependencies...$(NC)"
	@go mod download
	@go mod tidy
	@echo -e "$(YELLOW)Installing dashboard dependencies...$(NC)"
	@cd web/dashboard && npm install
	@echo -e "$(GREEN)✅ All dependencies installed$(NC)"
	@echo ""
	@echo -e "$(YELLOW)Bootstrapping configuration...$(NC)"
	@if [ ! -f deploy/config/pr-scout.yaml ]; then \
		echo -e "  $(RED)!$(NC) deploy/config/pr-scout.yaml not found — see README for template"; \
	else \
		echo -e "  $(YELLOW)-$(NC) deploy/config/pr-scout.yaml already exists"; \
	fi
	@if [ ! -f deploy/config/.env ]; then \
		echo -e "  $(RED)!$(NC) deploy/config/.env not found — see README for template"; \
	else \
		echo -e "  $(YELLOW)-$(NC) deploy/config/.env already exists"; \
	fi

# =============================================================================
# Dashboard
# =============================================================================

.PHONY: dashboard-build
dashboard-build: ## Build dashboard for production
	@echo -e "$(YELLOW)Building dashboard...$(NC)"
	@cd web/dashboard && npm run build
	@echo -e "$(GREEN)✅ Dashboard built to web/dashboard/dist/$(NC)"
