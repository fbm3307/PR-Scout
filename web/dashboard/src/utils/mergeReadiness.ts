import { parseJSON } from './parseJson.ts';
import type { PRWithReview, CIStatus, HumanReviewSummary } from '../types';

export function isMergeReady(pr: PRWithReview): boolean {
  const ci = parseJSON<CIStatus>(pr.ci_status);
  const review = parseJSON<HumanReviewSummary>(pr.human_review_summary);

  const ciGreen = ci?.overall_status === 'success';
  const requiredGreen = !ci || ci.required_total === 0 || ci.required_all_green;
  const hasApproval = (review?.approved_by?.length ?? 0) > 0;
  const noChangesRequested = (review?.changes_requested_by?.length ?? 0) === 0;
  const coderabbitClear = pr.coderabbit_total === 0 || pr.coderabbit_resolved >= pr.coderabbit_total;

  return ciGreen && requiredGreen && hasApproval && noChangesRequested && coderabbitClear;
}

export function getMergeBlockers(pr: PRWithReview): string[] {
  const ci = parseJSON<CIStatus>(pr.ci_status);
  const review = parseJSON<HumanReviewSummary>(pr.human_review_summary);
  const blockers: string[] = [];

  if (!ci || ci.overall_status !== 'success') {
    blockers.push('CI failing');
  }
  if (ci && ci.required_total > 0 && !ci.required_all_green) {
    blockers.push(`Required checks: ${ci.required_passed}/${ci.required_total} passing`);
  }
  if ((review?.approved_by?.length ?? 0) === 0) {
    blockers.push('No approvals');
  }
  if ((review?.changes_requested_by?.length ?? 0) > 0) {
    blockers.push(`${review!.changes_requested_by.length} changes requested`);
  }
  if (pr.coderabbit_total > 0 && pr.coderabbit_resolved < pr.coderabbit_total) {
    const unresolved = pr.coderabbit_total - pr.coderabbit_resolved;
    blockers.push(`${unresolved} unresolved CodeRabbit comment${unresolved > 1 ? 's' : ''}`);
  }

  return blockers;
}
