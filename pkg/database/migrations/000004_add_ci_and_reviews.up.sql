ALTER TABLE tracked_prs ADD COLUMN human_review_summary TEXT;
ALTER TABLE tracked_prs ADD COLUMN ci_status TEXT;
ALTER TABLE tracked_prs ADD COLUMN coderabbit_total INTEGER NOT NULL DEFAULT 0;
ALTER TABLE tracked_prs ADD COLUMN coderabbit_resolved INTEGER NOT NULL DEFAULT 0;
