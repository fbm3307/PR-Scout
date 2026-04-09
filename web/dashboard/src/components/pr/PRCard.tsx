import { useMemo } from 'react';
import { useNavigate } from 'react-router-dom';
import Card from '@mui/material/Card';
import CardActionArea from '@mui/material/CardActionArea';
import CardContent from '@mui/material/CardContent';
import Typography from '@mui/material/Typography';
import Box from '@mui/material/Box';
import Chip from '@mui/material/Chip';
import Stack from '@mui/material/Stack';
import { formatDistanceToNow } from 'date-fns';
import { parseJSON } from '../../utils/parseJson.ts';
import { HumanReviewChip, CIStatusChip, RequiredChecksChip, CodeRabbitChip, CodeRabbitSummaryChip } from './PRChips.tsx';
import { ReviewStatusBadge } from '../review/ReviewStatusBadge.tsx';
import type { PRWithReview, HumanReviewSummary, CIStatus } from '../../types';

interface PRCardProps {
  pr: PRWithReview;
}

export function PRCard({ pr }: PRCardProps) {
  const navigate = useNavigate();

  const reviewSummary = useMemo(
    () => parseJSON<HumanReviewSummary>(pr.human_review_summary),
    [pr.human_review_summary],
  );
  const ciStatus = useMemo(
    () => parseJSON<CIStatus>(pr.ci_status),
    [pr.ci_status],
  );

  const hasCodeRabbit = pr.coderabbit_total > 0 || !!pr.coderabbit_summary;
  const hasStatusRow = reviewSummary?.total_reviewers || ciStatus?.total_checks || hasCodeRabbit;

  return (
    <Card variant="outlined" sx={{ mb: 1 }}>
      <CardActionArea onClick={() => navigate(`/prs/${pr.id}`)}>
        <CardContent sx={{ py: 1.5, '&:last-child': { pb: 1.5 } }}>
          <Box sx={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between', gap: 1 }}>
            <Box sx={{ flex: 1, minWidth: 0 }}>
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 0.5 }}>
                <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 500 }}>
                  {pr.repo}
                </Typography>
                <Typography variant="body2" color="text.secondary">
                  #{pr.pr_number}
                </Typography>
                {pr.is_new && <Chip label="NEW" size="small" color="success" sx={{ height: 20, fontSize: '0.7rem' }} />}
                {pr.is_my_pr && <Chip label="MY PR" size="small" color="primary" variant="outlined" sx={{ height: 20, fontSize: '0.7rem' }} />}
              </Box>

              <Typography variant="subtitle1" sx={{ fontWeight: 500, lineHeight: 1.3 }}>
                {pr.title}
              </Typography>

              {pr.ai_summary && (
                <Typography variant="body2" color="text.secondary" sx={{ mt: 0.5 }} noWrap>
                  {pr.ai_summary.split('\n')[0]}
                </Typography>
              )}

              {hasStatusRow && (
                <Stack direction="row" spacing={0.5} sx={{ mt: 0.75, flexWrap: 'wrap', gap: 0.5 }}>
                  {reviewSummary && <HumanReviewChip summary={reviewSummary} />}
                  {ciStatus && <RequiredChecksChip ci={ciStatus} />}
                  {ciStatus && <CIStatusChip ci={ciStatus} />}
                  {pr.coderabbit_total > 0 ? (
                    <CodeRabbitChip total={pr.coderabbit_total} resolved={pr.coderabbit_resolved} />
                  ) : pr.coderabbit_summary ? (
                    <CodeRabbitSummaryChip summary={pr.coderabbit_summary} />
                  ) : null}
                </Stack>
              )}

              <Stack direction="row" spacing={1} sx={{ mt: 0.75, alignItems: 'center' }}>
                <Typography variant="caption" color="text.secondary">
                  {pr.author}
                </Typography>
                <Typography variant="caption" color="text.secondary">
                  {formatDistanceToNow(new Date(pr.created_at), { addSuffix: true })}
                </Typography>
                <Typography variant="caption" color="text.secondary">
                  +{pr.additions} -{pr.deletions}
                </Typography>
              </Stack>
            </Box>

            <Box sx={{ flexShrink: 0, pt: 0.5 }}>
              <ReviewStatusBadge review={pr.my_review} isMyPR={pr.is_my_pr} />
            </Box>
          </Box>
        </CardContent>
      </CardActionArea>
    </Card>
  );
}
