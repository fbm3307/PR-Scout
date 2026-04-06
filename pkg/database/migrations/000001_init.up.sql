-- Initial schema for pr-scout.
-- Uses common SQL subset compatible with both SQLite and PostgreSQL.

CREATE TABLE IF NOT EXISTS scan_runs (
    id            INTEGER PRIMARY KEY,
    started_at    TIMESTAMP NOT NULL,
    completed_at  TIMESTAMP,
    status        TEXT NOT NULL DEFAULT 'running',
    repos_scanned INTEGER NOT NULL DEFAULT 0,
    prs_found     INTEGER NOT NULL DEFAULT 0,
    new_prs       INTEGER NOT NULL DEFAULT 0,
    error_message TEXT
);

CREATE TABLE IF NOT EXISTS tracked_prs (
    id                 INTEGER PRIMARY KEY,
    scan_id            INTEGER NOT NULL REFERENCES scan_runs(id),
    repo               TEXT NOT NULL,
    pr_number          INTEGER NOT NULL,
    title              TEXT NOT NULL,
    body               TEXT,
    author             TEXT NOT NULL,
    url                TEXT NOT NULL,
    state              TEXT NOT NULL DEFAULT 'open',
    head_branch        TEXT,
    base_branch        TEXT,
    created_at         TIMESTAMP NOT NULL,
    updated_at         TIMESTAMP NOT NULL,
    labels             TEXT,
    is_new             INTEGER NOT NULL DEFAULT 0,
    ai_summary         TEXT,
    review_hints       TEXT,
    risk_notes         TEXT,
    coderabbit_summary TEXT,
    changed_files_count INTEGER NOT NULL DEFAULT 0,
    additions          INTEGER NOT NULL DEFAULT 0,
    deletions          INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_tracked_prs_scan_id ON tracked_prs(scan_id);
CREATE INDEX IF NOT EXISTS idx_tracked_prs_repo ON tracked_prs(repo);
CREATE INDEX IF NOT EXISTS idx_tracked_prs_state ON tracked_prs(state);
CREATE UNIQUE INDEX IF NOT EXISTS idx_tracked_prs_scan_repo_pr ON tracked_prs(scan_id, repo, pr_number);

CREATE TABLE IF NOT EXISTS review_comments (
    id        INTEGER PRIMARY KEY,
    pr_id     INTEGER NOT NULL REFERENCES tracked_prs(id),
    commenter TEXT NOT NULL,
    body      TEXT NOT NULL,
    file_path TEXT,
    line      INTEGER,
    created_at TIMESTAMP NOT NULL,
    is_bot    INTEGER NOT NULL DEFAULT 0,
    bot_name  TEXT,
    resolved  INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_review_comments_pr_id ON review_comments(pr_id);

CREATE TABLE IF NOT EXISTS my_review_status (
    id                   INTEGER PRIMARY KEY,
    pr_id                INTEGER NOT NULL REFERENCES tracked_prs(id),
    last_reviewed_at     TIMESTAMP,
    review_state         TEXT NOT NULL DEFAULT 'pending',
    status               TEXT NOT NULL DEFAULT 'waiting',
    commits_after_review INTEGER NOT NULL DEFAULT 0,
    unresolved_comments  INTEGER NOT NULL DEFAULT 0
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_my_review_status_pr_id ON my_review_status(pr_id);
