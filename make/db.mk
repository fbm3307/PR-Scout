# =============================================================================
# Database (PostgreSQL via Podman — only needed for postgres mode)
# =============================================================================

DB_CONTAINER := prscout-postgres
DB_PORT      := 5433
DB_USER      := your-db-user
DB_PASSWORD  := your-password-here
DB_NAME      := prscout

.PHONY: db-start
db-start: ## Start PostgreSQL container (for postgres mode)
	@if podman ps -a --format '{{.Names}}' | grep -q '^$(DB_CONTAINER)$$'; then \
		echo -e "$(YELLOW)Starting existing container...$(NC)"; \
		podman start $(DB_CONTAINER); \
	else \
		echo -e "$(YELLOW)Creating PostgreSQL container...$(NC)"; \
		podman run -d --name $(DB_CONTAINER) \
			-e POSTGRES_USER=$(DB_USER) \
			-e POSTGRES_PASSWORD=$(DB_PASSWORD) \
			-e POSTGRES_DB=$(DB_NAME) \
			-p $(DB_PORT):5432 \
			docker.io/library/postgres:16-alpine; \
	fi
	@echo -e "$(BLUE)PostgreSQL available at localhost:$(DB_PORT)$(NC)"

.PHONY: db-stop
db-stop: ## Stop PostgreSQL container
	@podman stop $(DB_CONTAINER) 2>/dev/null || true
	@echo -e "$(GREEN)✅ PostgreSQL stopped$(NC)"

.PHONY: db-clean
db-clean: db-stop ## Remove PostgreSQL container and data
	@podman rm $(DB_CONTAINER) 2>/dev/null || true
	@echo -e "$(GREEN)✅ PostgreSQL container removed$(NC)"
