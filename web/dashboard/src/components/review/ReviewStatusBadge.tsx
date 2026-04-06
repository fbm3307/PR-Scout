import Chip from '@mui/material/Chip';
import type { MyReviewStatus } from '../../types';

interface ReviewStatusBadgeProps {
  review?: MyReviewStatus;
  isMyPR?: boolean;
}

const reviewerStatusConfig: Record<string, { label: string; color: 'default' | 'success' | 'warning' | 'error' | 'info' }> = {
  needs_attention: { label: 'Needs Attention', color: 'warning' },
  waiting: { label: 'Waiting', color: 'default' },
  approved: { label: 'Approved', color: 'success' },
  merged: { label: 'Merged', color: 'info' },
  closed: { label: 'Closed', color: 'default' },
};

const authorStatusConfig: Record<string, { label: string; color: 'default' | 'success' | 'warning' | 'error' | 'info' }> = {
  needs_attention: { label: 'Reply to Reviews', color: 'warning' },
  waiting: { label: 'Awaiting Review', color: 'default' },
  approved: { label: 'Approved', color: 'success' },
  merged: { label: 'Merged', color: 'info' },
  closed: { label: 'Closed', color: 'default' },
};

export function ReviewStatusBadge({ review, isMyPR }: ReviewStatusBadgeProps) {
  if (!review) {
    return <Chip label={isMyPR ? 'Awaiting Review' : 'Not Reviewed'} size="small" variant="outlined" />;
  }

  const configMap = isMyPR ? authorStatusConfig : reviewerStatusConfig;
  const config = configMap[review.status] || { label: review.status, color: 'default' as const };

  return <Chip label={config.label} size="small" color={config.color} />;
}
