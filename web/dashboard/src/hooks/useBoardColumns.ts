import { useMemo } from 'react';
import { parseJSON } from '../utils/parseJson.ts';
import { isMergeReady } from '../utils/mergeReadiness.ts';
import type { PRWithReview, CIStatus } from '../types';

export interface BoardColumnData {
  id: string;
  label: string;
  prs: PRWithReview[];
  ciFailingCount: number;
  mergeReadyCount: number;
}

interface ColumnDef {
  id: string;
  label: string;
  match: (pr: PRWithReview) => boolean;
  sort: (a: PRWithReview, b: PRWithReview) => number;
}

const COLUMNS: ColumnDef[] = [
  {
    id: 'not_reviewed',
    label: 'Not Reviewed',
    match: (pr) => pr.state !== 'merged' && !pr.my_review,
    sort: (a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime(),
  },
  {
    id: 'needs_attention',
    label: 'Needs Attention',
    match: (pr) => pr.state !== 'merged' && pr.my_review?.status === 'needs_attention',
    sort: (a, b) => (b.my_review?.commits_after_review ?? 0) - (a.my_review?.commits_after_review ?? 0),
  },
  {
    id: 'waiting',
    label: 'Waiting',
    match: (pr) => pr.state !== 'merged' && pr.my_review?.status === 'waiting',
    sort: (a, b) => new Date(a.created_at).getTime() - new Date(b.created_at).getTime(),
  },
  {
    id: 'approved',
    label: 'Approved',
    match: (pr) => pr.state !== 'merged' && pr.my_review?.status === 'approved',
    sort: (a, b) => new Date(b.updated_at).getTime() - new Date(a.updated_at).getTime(),
  },
  {
    id: 'recently_merged',
    label: 'Recently Merged',
    match: (pr) => pr.state === 'merged',
    sort: (a, b) => new Date(b.updated_at).getTime() - new Date(a.updated_at).getTime(),
  },
];

export function useBoardColumns(prs: PRWithReview[]): BoardColumnData[] {
  return useMemo(() => {
    return COLUMNS.map((col) => {
      const matched = prs.filter(col.match).sort(col.sort);
      let ciFailingCount = 0;
      let mergeReadyCount = 0;

      for (const pr of matched) {
        const ci = parseJSON<CIStatus>(pr.ci_status);
        if (ci?.overall_status === 'failure') ciFailingCount++;
        if (isMergeReady(pr)) mergeReadyCount++;
      }

      return {
        id: col.id,
        label: col.label,
        prs: matched,
        ciFailingCount,
        mergeReadyCount,
      };
    });
  }, [prs]);
}
