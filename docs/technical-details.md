# PR Scout — Technical Details

This document provides an in-depth look at how PR Scout works under the hood — the scan pipeline, AI integration, review tracking logic, and deployment model.

## Overview

PR Scout is a daily PR review agent for the [codeready-toolchain](https://github.com/codeready-toolchain) GitHub organization. It scans all repositories for open pull requests and recently merged PRs, generates AI-powered review guidance via Anthropic Claude on Google Vertex AI, integrates CodeRabbit findings, and tracks your review comment lifecycle — all surfaced through a React + MUI web dashboard with a Kanban board view.

## Tech Stack

| Layer | Technology |
|-------|-----------|
| **Backend** | Go 1.25+, [Echo v5](https://echo.labstack.com/) HTTP framework |
| **Frontend** | React 19, [MUI 7](https://mui.com/), Vite 7, TypeScript |
| **Database** | SQLite (local) or PostgreSQL (shared), [golang-migrate](https://github.com/golang-migrate/migrate) for schema evolution |
| **GitHub** | [go-github v68](https://github.com/google/go-github) (REST), direct GraphQL for thread resolution |
| **AI/LLM** | [Anthropic SDK for Go](https://github.com/anthropics/anthropic-sdk-go) with Vertex AI authentication |
| **Config** | YAML (`gopkg.in/yaml.v3`) + `.env` files (`godotenv`) |
| **Containerization** | Multi-stage Dockerfile (Node → Go CGO → Alpine runtime) |

## Scan Pipeline

The scan is the core operation. It can be triggered three ways:
- `POST /api/v1/scan` (REST API)
- "Run Scan" button on the dashboard
- External cron job (`curl -s -X POST http://localhost:8080/api/v1/scan`)

### Step-by-step flow

1. **Scan initialization** — A new `scan_runs` row is inserted with status `running`.

2. **Repository discovery** — If `repos` is specified in configuration, only those repos are scanned. Otherwise, all non-archived, non-fork repositories in the org are fetched via `ListOrgRepos`.

3. **Concurrent PR scanning** — Up to **5 repositories are scanned in parallel** using Go's `errgroup`. For each repo, every open PR is processed:
   - **File collection** — PR file list and diff stats (additions, deletions, changed files count).
   - **Review comments** — Inline review comments fetched via `ListReviewComments`.
   - **Reviews** — Full review list fetched to build a human review summary (who approved, who requested changes) and to determine `MyReviewStatus`.
   - **CodeRabbit detection** — Comments authored by `coderabbitai[bot]` are identified from both PR review bodies and issue comments, then summarized into `coderabbit_summary` text.
   - **CI status** — Check runs on the head SHA are fetched, combined with branch protection required checks, to produce a CI status rollup.
   - **CodeRabbit thread resolution** — A GraphQL query counts total vs. resolved review threads authored by `coderabbitai[bot]`, stored as `coderabbit_total` / `coderabbit_resolved`.

4. **Review status computation** — `Tracker.ComputeReviewStatus` refines the initial review status:
   - If the PR is merged or closed, status reflects that.
   - If you approved the PR, it's marked accordingly.
   - If new commits were pushed after your last review, it's flagged as needing re-review.
   - Unresolved comment counts are attached.

5. **Recently merged PR scanning** — After open PRs are scanned, the scanner fetches PRs merged in the last **7 days** from each repo via `ListRecentlyMergedPRs` (uses GitHub's list-PRs API with `state=closed`, filtered by `merged_at`). Merged PRs get a lightweight scan:
   - **Review summary** — Human review summary is fetched to show who approved.
   - **Your review status** — `MyReviewStatus` is captured so the board can show "You approved" / "You reviewed" chips.
   - **Skipped** — File diffs, CI checks, CodeRabbit analysis, and LLM processing are skipped since the PR is already merged.
   - Merged PRs are stored with `state = "merged"` and `llm_status = "skipped"`.

6. **Persistence** — All `tracked_prs`, `review_comments`, and `my_review_status` rows are written to the database. The scan is marked `completed` with final counts.

7. **LLM analysis (async)** — A background goroutine (`llmWorkerLoop`) continuously polls for PRs with `llm_status = 'pending'`. For each:
   - PR files are re-fetched from GitHub to get diff content.
   - The diff is truncated to ~30,000 characters to fit within the LLM context window.
   - `AnalyzePR` sends the diff + PR metadata to Claude via the Anthropic SDK.
   - Claude returns a structured response with three fields: **AI summary**, **review hints** (key areas to focus on), and **risk notes** (potential issues).
   - Results are persisted back to `tracked_prs`. On failure, the row remains `pending` for retry.

### What gets skipped

- **Bot-authored PRs** (e.g., Dependabot) — skipped for LLM analysis
- **Draft / WIP PRs** — skipped for LLM analysis
- **Old PRs** — PRs older than `max_pr_age_days` are skipped for LLM analysis
- **Unchanged PRs** — If a PR's `updated_at` hasn't changed since the last scan, prior AI text is reused

## AI Integration

PR Scout uses **Anthropic Claude** (currently `claude-sonnet-4@20250514`) hosted on **Google Vertex AI**.

### Authentication

The LLM client uses Google Application Default Credentials (`gcloud auth application-default login`). The Vertex project ID is read from the environment variable specified by `llm.project_id_env` (default: `ANTHROPIC_VERTEX_PROJECT_ID`).

### Prompt design

Two prompt variants exist:
- **Reviewer prompt** — When the PR is authored by someone else: focuses on what to look for during review, potential risks, and areas that need careful attention.
- **Author prompt** — When `is_my_pr` is true: focuses on what reviewers might flag, suggestions for the PR description, and self-review checklist items.

The prompt includes:
- PR title, author, description/body
- Full file diff (truncated to ~30k characters)
- File change statistics
- Labels and draft status

### Structured output

Claude's response is parsed into three distinct sections:
- **`ai_summary`** — A concise summary of what the PR does
- **`review_hints`** — Specific areas a reviewer should focus on
- **`risk_notes`** — Potential risks, edge cases, or concerns

## CodeRabbit Integration

[CodeRabbit](https://coderabbit.ai/) is an AI code review bot. PR Scout doesn't replace it — it **aggregates and surfaces** its findings alongside your own review context.

- **Detection** — Any review or issue comment authored by `coderabbitai[bot]` is identified.
- **Summarization** — `SummarizeCodeRabbitFindings` extracts and condenses CodeRabbit's review text into a readable summary.
- **Thread resolution** — A GitHub GraphQL query counts how many CodeRabbit review threads have been resolved vs. total, giving a quick "5/8 resolved" metric.
- **Dashboard display** — CodeRabbit findings appear in the PR detail view alongside human review comments and AI analysis.

## Review Tracking

PR Scout tracks **your** interaction with each PR:

| Status | Meaning |
|--------|---------|
| `needs_review` | You haven't reviewed this PR yet |
| `reviewed` | You submitted a review, no changes since |
| `needs_re_review` | New commits were pushed after your last review |
| `approved` | You approved the PR |
| `changes_requested` | You requested changes |
| `merged` | The PR was merged |
| `closed` | The PR was closed without merging |

For PRs **you authored** (`is_my_pr`), the logic flips — it tracks whether reviewers have responded, whether you have pending review requests, and whether comment threads were addressed.

## Database

### Dual driver support

A single config field (`database.driver`) switches between:
- **SQLite** — Zero-setup, file-based, ideal for local/single-user use. Uses `github.com/mattn/go-sqlite3` (requires CGO).
- **PostgreSQL** — Connection-pooled, suitable for shared/team deployments. Uses `github.com/jackc/pgx/v5/stdlib`. Connection parameters can be overridden via `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `DB_SSLMODE` environment variables.

### Migrations

Schema evolution is handled by [golang-migrate](https://github.com/golang-migrate/migrate) with SQL migrations embedded in the Go binary:

| Migration | Description |
|-----------|-------------|
| `000001` | Core tables: `scan_runs`, `tracked_prs`, `review_comments`, `my_review_status` |
| `000002` | Add `is_my_pr` column to `tracked_prs` |
| `000003` | Add LLM columns: `ai_summary`, `review_hints`, `risk_notes`, `llm_status` |
| `000004` | Add CI status, human review summary, CodeRabbit total/resolved counts |

## Configuration

All configuration lives in `deploy/config/pr-scout.yaml` with secrets in `deploy/config/.env`.

### Configuration reference

```yaml
github:
  org: codeready-toolchain        # GitHub org to scan
  username: your-username          # Your GitHub username (for review tracking)
  token_env: GITHUB_TOKEN          # Env var name holding the PAT
  repos: []                        # Explicit repo list (empty = scan all)
  max_prs_per_repo: 50             # Max open PRs to process per repo

llm:
  enabled: true                    # Enable/disable AI analysis
  provider: anthropic-vertex       # LLM provider
  model: claude-sonnet-4@20250514  # Model identifier
  project_id_env: ANTHROPIC_VERTEX_PROJECT_ID
  region: us-east5                 # Vertex AI region
  max_pr_age_days: 30              # Skip LLM for PRs older than this

database:
  driver: sqlite                   # sqlite or postgres
  # PostgreSQL settings (or use DB_* env vars):
  host: localhost
  port: 5432
  user: prscout
  password: prscout
  dbname: prscout
  sslmode: disable

server:
  port: 8080                       # Backend listen port
  dashboard_url: http://localhost:5173  # CORS origin for dashboard
```

### Environment variables

| Variable | Purpose |
|----------|---------|
| `GITHUB_TOKEN` | GitHub personal access token (`repo` + `read:org` scopes) |
| `ANTHROPIC_VERTEX_PROJECT_ID` | Google Cloud project for Vertex AI |
| `CONFIG_DIR` | Override config directory (default: `./deploy/config`) |
| `DASHBOARD_DIR` | Path to built dashboard assets (for production) |
| `LOG_LEVEL` | Logging verbosity: `debug`, `info` (default), `warn`, `error` |
| `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `DB_SSLMODE` | PostgreSQL connection overrides |

## Deployment

### Local development

```bash
make setup    # Install Go + Node dependencies
make dev      # Start backend (:8080) + dashboard (:5173) concurrently
make scan     # Trigger a scan via curl
```

The Vite dev server proxies `/api` and `/health` requests to the Go backend.

### Production (Docker)

The multi-stage Dockerfile produces a single container:

1. **Node stage** — Builds the React dashboard (`npm run build`)
2. **Go stage** — Compiles the Go binary with CGO enabled (required for SQLite)
3. **Runtime stage** — Alpine-based, serves the dashboard as static assets embedded in the Go server

```bash
docker build -t pr-scout .
docker run -p 8080:8080 \
  -e GITHUB_TOKEN=... \
  -e ANTHROPIC_VERTEX_PROJECT_ID=... \
  -v ./data:/data \
  pr-scout
```

### Scheduling

PR Scout is designed to run continuously as a server, with scans triggered externally:

```bash
# Cron: scan at 8:00 AM daily
0 8 * * * curl -s -X POST http://localhost:8080/api/v1/scan
```

## Kanban Board

The dashboard supports two view modes — a traditional **list view** and a **Kanban board** — toggled via a button group in the tabs row. The selected mode is persisted to `localStorage`.

### Board columns

The board groups non-authored PRs (`!is_my_pr`) into five columns based on review status:

| Column | Match criteria | Sort order |
|--------|---------------|------------|
| **Not Reviewed** | No `my_review` record, state is not `merged` | Newest first |
| **Needs Attention** | `my_review.status === 'needs_attention'` | Most commits-after-review first |
| **Waiting** | `my_review.status === 'waiting'` | Oldest first |
| **Approved** | `my_review.status === 'approved'` | Most recently updated first |
| **Recently Merged** | `state === 'merged'` | Most recently merged first |

Column headers show a count chip and a green "N ready" chip when merge-ready PRs exist.

### Board cards

Each card is a compact `BoardCard` component showing:
- Repo name, PR number, and status chips: "NEW", "DRAFT", "STALE", or "MERGED"
- Title (2-line CSS clamp)
- Status chips: human review, required checks, CI, CodeRabbit (reused from shared `PRChips`; CodeRabbit uses the actual CodeRabbit avatar icon)
- Author avatar (24px GitHub avatar), reviewer avatar group (max 3, 20px), age, and `+additions -deletions`
- For merged PRs: "You approved" (green) or "You reviewed" (neutral) chip when you participated
- Draft PRs render at 65% opacity with a gray left border
- Stale PRs (no activity for 90+ days) show a warning-colored "STALE" chip with an orange left border

### Recently Merged column

The "Recently Merged" column includes a **1d / 3d / 7d** toggle button group in its header to filter by merge recency. Defaults to 7d.

### Stale PR handling

In each open-PR column (Not Reviewed, Needs Attention, Waiting, Approved), PRs with `updated_at` older than 90 days are grouped at the bottom behind a collapsed "Show N stale" expander. The column header shows the active count with a muted "+N stale" suffix. This keeps the board focused on actionable PRs.

### Draft PR handling

PRs flagged as `is_draft` (from GitHub's draft status or WIP title prefixes) show a "DRAFT" chip, render at reduced opacity (65%), and have a gray left border. Draft status is detected in the backend scanner and stored in the `tracked_prs` table.

### GitHub avatars

Author and reviewer avatars are displayed on both board cards and list cards using GitHub's predictable avatar URL pattern (`https://github.com/{username}.png`). Bot accounts like `dependabot[bot]` are handled by stripping the `[bot]` suffix (e.g., `https://github.com/dependabot.png`). No backend or database changes were needed — the existing `author` field and `HumanReviewSummary` reviewer lists provide the usernames.

### Merge-readiness detection

A PR is considered merge-ready when all conditions are met:
- CI overall status is `success`
- All required checks pass (or no required checks configured)
- At least one human approval
- No outstanding change requests
- All CodeRabbit comments resolved (or no CodeRabbit comments)

Merge-ready cards get a **4px green left border** (`success.main`). Non-ready cards show **merge blockers on hover** (e.g., "CI failing", "No approvals", "2 unresolved CodeRabbit comments").

Merged cards get a **4px blue left border** (`info.main`) instead.

### View mode interactions

- The "My Review Status" filter is hidden in board mode (the board's columns serve the same purpose).
- Switching to board mode clears any active review status filter.
- The "My PRs" tab always shows the list view; the board toggle is disabled on that tab.
- The list view filters out merged PRs (they only appear on the board).
- The `Container` uses `maxWidth={false}` (full width) in board mode with minimal padding and `overflow: hidden` to contain the scrollable columns within the viewport.

## Design Decisions

| Decision | Rationale |
|----------|-----------|
| **Read-only on GitHub** | PR Scout never posts comments or reviews to GitHub — it's purely a consumer. This avoids permission concerns and noise in PR threads. |
| **Async LLM processing** | AI analysis runs in a background worker, not inline with the scan. This keeps scans fast and lets the LLM process at its own pace. |
| **Dual database** | SQLite for zero-friction local setup; PostgreSQL for teams who want a shared instance. Same schema, same queries. |
| **Diff truncation** | PR diffs are capped at ~30k characters before sending to Claude. This balances context quality with token limits and cost. |
| **Reuse prior AI text** | If a PR hasn't changed since the last scan, its AI analysis is carried forward. This saves LLM calls and reduces latency. |
| **CodeRabbit as complement** | Rather than replacing CodeRabbit, PR Scout surfaces its findings alongside its own AI analysis, giving reviewers a unified view. |
| **Kanban board is reviewer-centric** | The board groups PRs by your review status, not by repo or author. Authored PRs are excluded since the "My PRs" tab handles those. |
| **Lightweight merged PR scan** | Merged PRs skip CI, CodeRabbit, files, and LLM analysis — only review summary and your review status are fetched. This minimizes GitHub API calls while providing enough data for the board. |
| **Client-side merge-readiness** | Merge-readiness is computed in the browser from existing API data (CI status, review summary, CodeRabbit counts) rather than adding a backend field, keeping the API unchanged. |
| **7-day merged window** | Recently merged PRs older than 7 days are dropped from the board to keep the column focused and scannable. |
| **90-day stale cutoff** | PRs with no update in 90 days are collapsed at the bottom of each column. They're not hidden — just deprioritized behind an expander. |
| **GitHub avatars without backend changes** | Avatar URLs are derived from usernames (`github.com/{user}.png`), avoiding additional API calls or database fields. Bot accounts are handled by stripping `[bot]`. |
| **Draft/stale as visual indicators** | Draft and stale status are shown as chips and border colors rather than separate columns, keeping the board layout focused on review workflow. |
