export interface ScanRun {
  id: number;
  started_at: string;
  completed_at?: string;
  status: string;
  repos_scanned: number;
  prs_found: number;
  new_prs: number;
  error_message?: string;
}

export interface HumanReviewSummary {
  total_reviewers: number;
  approved_by: string[];
  changes_requested_by: string[];
  commented_by: string[];
}

export interface CIStatus {
  overall_status: 'success' | 'failure' | 'pending' | 'mixed';
  total_checks: number;
  passed: number;
  failed: number;
  pending: number;
  required_all_green: boolean;
  required_total: number;
  required_passed: number;
  failed_checks: { name: string; conclusion: string; summary?: string }[];
}

export interface TrackedPR {
  id: number;
  scan_id: number;
  repo: string;
  pr_number: number;
  title: string;
  body?: string;
  author: string;
  url: string;
  state: string;
  head_branch: string;
  base_branch: string;
  created_at: string;
  updated_at: string;
  labels?: string;
  is_new: boolean;
  is_my_pr: boolean;
  is_draft: boolean;
  ai_summary?: string;
  review_hints?: string;
  risk_notes?: string;
  coderabbit_summary?: string;
  human_review_summary?: string;
  ci_status?: string;
  coderabbit_total: number;
  coderabbit_resolved: number;
  changed_files_count: number;
  additions: number;
  deletions: number;
}

export interface MyReviewStatus {
  id: number;
  pr_id: number;
  last_reviewed_at?: string;
  review_state: string;
  status: string;
  commits_after_review: number;
  unresolved_comments: number;
}

export interface PRWithReview extends TrackedPR {
  my_review?: MyReviewStatus;
}

export interface ReviewComment {
  id: number;
  pr_id: number;
  commenter: string;
  body: string;
  file_path?: string;
  line?: number;
  created_at: string;
  is_bot: boolean;
  bot_name?: string;
  resolved: boolean;
}

export interface Digest {
  scan?: ScanRun;
  total_open_prs: number;
  new_prs: number;
  needs_attention: number;
  repos_with_activity: number;
  top_repos: RepoStat[];
}

export interface RepoStat {
  repo: string;
  pr_count: number;
}

export interface ListResponse<T> {
  items: T[];
  total: number;
  page: number;
  per_page: number;
}
