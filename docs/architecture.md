# PR Scout — Architecture

## High-Level Overview

```mermaid
flowchart TB
    subgraph Trigger["Trigger"]
        CRON["Cron / systemd timer"]
        UI_BTN["Dashboard<br/>'Run Scan' button"]
    end

    subgraph Backend["Go Backend — Echo v5 · :8080"]
        API["API Layer<br/>pkg/api/"]
        SCAN["Scan Service<br/>pkg/services/scan_service.go"]
        PR_SVC["PR Service<br/>pkg/services/pr_service.go"]
        DIGEST["Digest Service<br/>pkg/services/digest_service.go"]
        SCANNER["GitHub Scanner<br/>pkg/github/scanner.go"]
        TRACKER["Review Tracker<br/>pkg/github/tracker.go"]
        CR["CodeRabbit Parser<br/>pkg/github/coderabbit.go"]
        GQL["GraphQL Client<br/>pkg/github/graphql.go"]
        LLM["LLM Client<br/>pkg/llm/"]
        CFG["Config Loader<br/>pkg/config/"]
        DB_CLIENT["Database Client<br/>pkg/database/"]
    end

    subgraph External["External Services"]
        GH_REST["GitHub REST API<br/>PRs · Reviews · Comments<br/>Files · Check Runs"]
        GH_GQL["GitHub GraphQL API<br/>CodeRabbit thread resolution"]
        VERTEX["Google Vertex AI<br/>Anthropic Claude"]
    end

    subgraph Storage["Storage"]
        SQLITE[("SQLite<br/>Local dev")]
        POSTGRES[("PostgreSQL<br/>Shared deployments")]
    end

    subgraph Dashboard["React Dashboard — Vite · :5173"]
        DASH_PAGE["Dashboard Page<br/>Digest cards + PR list<br/>+ Kanban board"]
        BOARD["Kanban Board<br/>Not Reviewed · Needs Attention<br/>Waiting · Approved · Recently Merged"]
        DETAIL_PAGE["PR Detail Page<br/>AI summary · Comments<br/>Review status"]
        API_CLIENT["Axios API Client<br/>→ /api/v1/*"]
        DASH_PAGE --> BOARD
    end

    CRON -->|"POST /api/v1/scan"| API
    UI_BTN --> API_CLIENT
    API_CLIENT -->|"proxy :5173 → :8080"| API

    API --> SCAN
    API --> PR_SVC
    API --> DIGEST

    SCAN --> SCANNER
    SCAN -->|"async worker"| LLM
    SCANNER --> GH_REST
    SCANNER --> CR
    SCANNER --> GQL
    GQL --> GH_GQL
    SCANNER --> TRACKER
    LLM --> VERTEX

    SCAN --> DB_CLIENT
    PR_SVC --> DB_CLIENT
    DIGEST --> DB_CLIENT
    DB_CLIENT --> SQLITE
    DB_CLIENT --> POSTGRES

    PR_SVC --> DASH_PAGE
    PR_SVC --> DETAIL_PAGE

    style Trigger fill:#fef3c7,stroke:#d97706
    style Backend fill:#dbeafe,stroke:#2563eb
    style External fill:#fce7f3,stroke:#db2777
    style Storage fill:#d1fae5,stroke:#059669
    style Dashboard fill:#ede9fe,stroke:#7c3aed
```

## Scan Data Flow

End-to-end sequence from triggering a scan to rendering results on the dashboard:

```mermaid
sequenceDiagram
    participant User as Operator / Dashboard
    participant API as Echo API (:8080)
    participant Scan as ScanService
    participant Scanner as GitHub Scanner
    participant GH as GitHub REST + GraphQL
    participant DB as SQLite / PostgreSQL
    participant LLM as Claude (Vertex AI)
    participant Dash as React Dashboard

    User->>API: POST /api/v1/scan
    API->>Scan: RunScan()

    rect rgb(147, 197, 253)
        Note over Scan,GH: Parallel repo scanning (up to 5 concurrent)
        Scan->>Scanner: ScanRepo (per repo)
        Scanner->>GH: List open PRs
        Scanner->>GH: Fetch files, reviews, comments
        Scanner->>GH: Fetch check runs + branch protection
        Scanner->>GH: GraphQL — CodeRabbit thread resolution
        GH-->>Scanner: PR data + review threads + CI status
    end

    Scanner-->>Scan: []TrackedPR + ReviewComments

    rect rgb(253, 224, 137)
        Note over Scan,GH: Recently merged PRs (last 7 days)
        Scan->>Scanner: ScanMergedRepo (per repo)
        Scanner->>GH: List closed PRs (merged_at > 7 days ago)
        Scanner->>GH: Fetch reviews (lightweight)
        GH-->>Scanner: Merged PR data + review summary
    end

    Scanner-->>Scan: []TrackedPR (state=merged)

    rect rgb(134, 239, 172)
        Note over Scan,DB: Persist scan results
        Scan->>DB: INSERT scan_runs (completed)
        Scan->>DB: INSERT tracked_prs
        Scan->>DB: INSERT review_comments
        Scan->>DB: UPSERT my_review_status
    end

    rect rgb(249, 168, 212)
        Note over Scan,LLM: Async LLM worker (background)
        Scan->>DB: Poll rows with llm_status = 'pending'
        Scan->>GH: Fetch PR files (for diff context)
        Scan->>LLM: AnalyzePR (diff + metadata)
        LLM-->>Scan: AI summary, review hints, risk notes
        Scan->>DB: UPDATE ai_summary, review_hints, risk_notes
    end

    User->>Dash: Open dashboard
    Dash->>API: GET /api/v1/digest
    Dash->>API: GET /api/v1/prs
    API->>DB: Query latest completed scan
    DB-->>API: Results
    API-->>Dash: JSON responses
    Dash-->>User: Render digest cards + PR list / Kanban board + detail views
```

## Package Structure

```
pr-scout/
├── cmd/pr-scout/
│   └── main.go              ← Entry point: wiring, server start, graceful shutdown
├── pkg/
│   ├── api/
│   │   ├── server.go        ← Echo router, CORS, static SPA serving
│   │   ├── handler_scan.go  ← POST /api/v1/scan
│   │   ├── handler_prs.go   ← GET /api/v1/prs, /api/v1/prs/:id
│   │   ├── handler_digest.go← GET /api/v1/digest
│   │   └── responses.go     ← Shared JSON response types
│   ├── config/
│   │   └── config.go        ← YAML + .env loading, defaults, DB env overrides
│   ├── database/
│   │   ├── client.go        ← SQLite/PostgreSQL connect + golang-migrate
│   │   └── migrations/      ← Embedded SQL migrations (4 versions)
│   ├── github/
│   │   ├── client.go        ← go-github OAuth2 client wrapper + ListRecentlyMergedPRs
│   │   ├── scanner.go       ← Per-repo PR scanning, merged PR scanning
│   │   ├── tracker.go       ← MyReviewStatus computation (post-scan)
│   │   ├── coderabbit.go    ← CodeRabbit bot comment filtering + summarization
│   │   └── graphql.go       ← Direct GraphQL for thread resolution counts
│   ├── llm/
│   │   ├── client.go        ← Anthropic SDK + Vertex auth, AnalyzePR
│   │   └── prompts.go       ← System/user prompt templates, diff truncation
│   ├── models/
│   │   └── *.go             ← TrackedPR, ReviewComment, ScanRun, CIStatus, etc.
│   └── services/
│       ├── scan_service.go  ← RunScan orchestration, merged PR scanning, LLM worker
│       ├── pr_service.go    ← Read APIs (list/get PRs from latest scan)
│       └── digest_service.go← Aggregate stats for digest endpoint
├── web/dashboard/
│   └── src/
│       ├── App.tsx           ← React Router (/, /prs/:id)
│       ├── pages/            ← DashboardPage (list + Kanban board), PRDetailPage
│       ├── components/
│       │   ├── board/        ← BoardView, BoardColumn, BoardCard (Kanban)
│       │   ├── digest/       ← DigestCards
│       │   ├── pr/           ← PRCard, PRList, PRFilters, PRChips (shared chips)
│       │   ├── review/       ← ReviewStatusBadge
│       │   └── shared/       ← Reusable UI components
│       ├── hooks/
│       │   └── useBoardColumns.ts  ← Kanban column grouping, sorting, aggregates
│       ├── utils/
│       │   ├── parseJson.ts        ← Safe JSON parsing
│       │   └── mergeReadiness.ts   ← isMergeReady, getMergeBlockers
│       ├── services/api.ts   ← Axios client to /api/v1
│       ├── types/index.ts    ← TypeScript interfaces
│       └── theme/index.ts    ← MUI theme (light/dark)
├── deploy/config/
│   ├── pr-scout.yaml         ← Main configuration
│   └── .env                  ← Secrets (gitignored)
├── Dockerfile                ← Multi-stage: Node + Go + Alpine runtime
└── Makefile                  ← dev, build, scan, test, lint, db-start
```

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/health` | Health check |
| `POST` | `/api/v1/scan` | Trigger a new org-wide scan |
| `GET` | `/api/v1/prs` | List PRs from latest scan (filterable) |
| `GET` | `/api/v1/prs/:id` | Get single PR with comments |
| `GET` | `/api/v1/my-reviews` | PRs where you are a reviewer |
| `GET` | `/api/v1/digest` | Aggregated digest stats |

## Database Schema (Simplified)

```mermaid
erDiagram
    scan_runs {
        int id PK
        string status
        timestamp started_at
        timestamp completed_at
        int total_prs
        int new_prs
    }

    tracked_prs {
        int id PK
        int scan_run_id FK
        string repo
        int pr_number
        string title
        string author
        string state
        bool is_draft
        bool is_my_pr
        string ai_summary
        string review_hints
        string risk_notes
        string llm_status
        string ci_status
        string coderabbit_summary
        int coderabbit_total
        int coderabbit_resolved
        string human_review_summary
    }

    review_comments {
        int id PK
        int tracked_pr_id FK
        string author
        string body
        string path
        int line
        bool is_coderabbit
        bool resolved
    }

    my_review_status {
        int id PK
        int tracked_pr_id FK
        string status
        int unresolved_count
        timestamp last_review_at
    }

    scan_runs ||--o{ tracked_prs : "has"
    tracked_prs ||--o{ review_comments : "has"
    tracked_prs ||--o| my_review_status : "has"
```
