# PR Scout

Daily PR review agent for the [codeready-toolchain](https://github.com/codeready-toolchain) GitHub org. Scans all repositories for open pull requests, generates AI-powered review guidance, integrates CodeRabbit findings, and tracks your review comment lifecycle.

## Features

- **Org-wide PR scanning** — scans all repos in the GitHub org on a daily schedule
- **Kanban review board** — visual board with columns for Not Reviewed, Needs Attention, Waiting, Approved, and Recently Merged PRs. Toggle between board and list views; preference persists across sessions.
- **Merge-readiness detection** — composite signal (CI green + required checks + approvals + no changes requested + CodeRabbit clear) with a green border on merge-ready cards and blockers shown on hover
- **Recently merged PR tracking** — merged PRs from the last 7 days appear on the board with "MERGED" and "You approved" / "You reviewed" chips so you can close the review loop
- **AI review guidance** — generates PR summaries, key review areas, and risk notes via Anthropic Claude (Vertex AI)
- **CodeRabbit integration** — parses and surfaces CodeRabbit bot review comments
- **Review tracking** — monitors your review comments: were they addressed? new commits since your review? PR merged/closed?
- **Web dashboard** — React + MUI dashboard showing digest stats, filterable PR list, detailed PR views, and Kanban board
- **Dual database support** — SQLite for local use, PostgreSQL for shared deployments

## Quick Start

```bash
# 1. Install dependencies
make setup

# 2. Configure
cat > deploy/config/pr-scout.yaml <<EOF
github:
  org: your-github-org
  username: your-github-username
  token_env: GITHUB_TOKEN

llm:
  enabled: true
  provider: anthropic-vertex
  model: claude-sonnet-4@20250514
  project_id_env: ANTHROPIC_VERTEX_PROJECT_ID
  region: us-east5

database:
  driver: sqlite

server:
  port: 8080
  dashboard_url: http://localhost:5173
EOF

cat > deploy/config/.env <<EOF
GITHUB_TOKEN=your-github-token
ANTHROPIC_VERTEX_PROJECT_ID=your-project-id
EOF

# 3. Start
make dev

# 4. Trigger a scan
make scan
# or visit http://localhost:5173 and click "Run Scan"
```

## Prerequisites

- Go 1.25+
- Node.js 22+
- A GitHub personal access token with `repo` + `read:org` scopes

Optional:
- `ANTHROPIC_VERTEX_PROJECT_ID` + `gcloud auth application-default login` for AI summaries
- Podman/Docker for PostgreSQL mode (`make db-start`)

## Configuration

Key settings in `deploy/config/pr-scout.yaml`:
- `github.org` — GitHub organization to scan
- `github.username` — your GitHub username (for review tracking)
- `llm.enabled` — enable/disable AI summaries
- `database.driver` — `sqlite` (default) or `postgres`

## Daily Scheduling

Add a cron entry to scan every morning:

```bash
# Run scan at 8:00 AM daily
0 8 * * * cd /path/to/pr-scout && curl -s -X POST http://localhost:8080/api/v1/scan
```

Or use a systemd timer for more control. The backend must be running for the cron `curl` to work. Alternatively, you can run the scan as part of `make dev` startup.

## Architecture

```
Go Backend (Echo v5, :8080)
├── pkg/api/        — HTTP handlers
├── pkg/config/     — YAML + env config
├── pkg/database/   — SQLite/PostgreSQL + migrations
├── pkg/github/     — GitHub API client, scanner, CodeRabbit parser, review tracker
├── pkg/llm/        — Anthropic Claude via Vertex AI
├── pkg/models/     — Data types
└── pkg/services/   — Scan, PR, and Digest services

React Dashboard (Vite, :5173)
└── web/dashboard/src/
    ├── pages/      — DashboardPage, PRDetailPage
    ├── components/ — digest/, pr/, board/, review/, shared/
    ├── hooks/      — useBoardColumns (Kanban grouping/sorting)
    ├── utils/      — parseJson, mergeReadiness
    ├── services/   — API client
    └── types/      — TypeScript interfaces
```

For the full architecture with diagrams and data-flow sequences, see [Architecture](docs/architecture.md).

For an in-depth look at the scan pipeline, AI integration, review tracking, database design, and deployment model, see [Technical Details](docs/technical-details.md).

## Development

```bash
make help          # Show all targets
make doctor        # Check prerequisites
make dev           # Start backend + dashboard
make build         # Build Go binary
make scan          # Trigger a scan
make test          # Run all tests
make lint          # Run linters
make dashboard-build  # Build dashboard for production
make db-start      # Start PostgreSQL (for postgres mode)
```

## License

See [LICENSE](LICENSE).
