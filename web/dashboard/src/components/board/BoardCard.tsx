import { useMemo } from 'react';
import { useNavigate } from 'react-router-dom';
import Card from '@mui/material/Card';
import CardActionArea from '@mui/material/CardActionArea';
import CardContent from '@mui/material/CardContent';
import Typography from '@mui/material/Typography';
import Box from '@mui/material/Box';
import Chip from '@mui/material/Chip';
import Stack from '@mui/material/Stack';
import Tooltip from '@mui/material/Tooltip';
import Avatar from '@mui/material/Avatar';
import AvatarGroup from '@mui/material/AvatarGroup';
import MergeIcon from '@mui/icons-material/MergeType';
import CheckIcon from '@mui/icons-material/Check';
import VisibilityIcon from '@mui/icons-material/Visibility';
import EditNoteIcon from '@mui/icons-material/EditNote';
import HistoryIcon from '@mui/icons-material/History';
import { formatDistanceToNow, subDays } from 'date-fns';
import { parseJSON } from '../../utils/parseJson.ts';
import { ghAvatar } from '../../utils/ghAvatar.ts';
import { isMergeReady, getMergeBlockers } from '../../utils/mergeReadiness.ts';
import { HumanReviewChip, CIStatusChip, RequiredChecksChip, CodeRabbitChip, CodeRabbitSummaryChip } from '../pr/PRChips.tsx';
import type { PRWithReview, HumanReviewSummary, CIStatus } from '../../types';

function getReviewerUsernames(summary: HumanReviewSummary | null): string[] {
  if (!summary) return [];
  const set = new Set<string>();
  for (const u of summary.approved_by ?? []) set.add(u);
  for (const u of summary.changes_requested_by ?? []) set.add(u);
  for (const u of summary.commented_by ?? []) set.add(u);
  return [...set];
}

interface BoardCardProps {
  pr: PRWithReview;
}

export function BoardCard({ pr }: BoardCardProps) {
  const navigate = useNavigate();
  const isMerged = pr.state === 'merged';

  const reviewSummary = useMemo(
    () => parseJSON<HumanReviewSummary>(pr.human_review_summary),
    [pr.human_review_summary],
  );
  const ciStatus = useMemo(
    () => parseJSON<CIStatus>(pr.ci_status),
    [pr.ci_status],
  );

  const reviewerUsernames = useMemo(() => getReviewerUsernames(reviewSummary), [reviewSummary]);

  const isStale = !isMerged && new Date(pr.updated_at) < subDays(new Date(), 90);
  const mergeReady = !isMerged && isMergeReady(pr);
  const tooltipText = isMerged
    ? `Merged ${formatDistanceToNow(new Date(pr.updated_at), { addSuffix: true })}`
    : mergeReady
      ? 'Ready to merge'
      : getMergeBlockers(pr).join(', ');

  const hasCodeRabbit = pr.coderabbit_total > 0 || !!pr.coderabbit_summary;
  const hasStatusRow = reviewSummary?.total_reviewers || ciStatus?.total_checks || hasCodeRabbit;

  const borderColor = isMerged ? 'info.main' : pr.is_draft ? 'text.disabled' : isStale ? 'warning.main' : mergeReady ? 'success.main' : undefined;

  return (
    <Tooltip title={tooltipText} arrow placement="right" enterDelay={400}>
      <Card
        variant="outlined"
        sx={{
          mb: 1,
          borderLeft: borderColor ? '4px solid' : undefined,
          borderLeftColor: borderColor,
          opacity: pr.is_draft ? 0.65 : 1,
        }}
      >
        <CardActionArea onClick={() => navigate(`/prs/${pr.id}`)}>
          <CardContent sx={{ py: 1, px: 1.5, '&:last-child': { pb: 1 } }}>
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5, mb: 0.25 }}>
              <Typography variant="caption" color="text.secondary" sx={{ fontWeight: 500 }} noWrap>
                {pr.repo}
              </Typography>
              <Typography variant="caption" color="text.secondary">
                #{pr.pr_number}
              </Typography>
              {pr.is_draft && (
                <Chip
                  icon={<EditNoteIcon sx={{ fontSize: 12 }} />}
                  label="DRAFT"
                  size="small"
                  sx={{ height: 18, fontSize: '0.65rem', bgcolor: 'action.selected' }}
                />
              )}
              {isStale && (
                <Chip
                  icon={<HistoryIcon sx={{ fontSize: 12 }} />}
                  label="STALE"
                  size="small"
                  color="warning"
                  variant="outlined"
                  sx={{ height: 18, fontSize: '0.65rem' }}
                />
              )}
              {isMerged && (
                <Chip
                  icon={<MergeIcon sx={{ fontSize: 12 }} />}
                  label="MERGED"
                  size="small"
                  color="info"
                  sx={{ height: 18, fontSize: '0.65rem' }}
                />
              )}
              {isMerged && pr.my_review && (
                pr.my_review.review_state === 'approved' ? (
                  <Chip
                    icon={<CheckIcon sx={{ fontSize: 12 }} />}
                    label="You approved"
                    size="small"
                    color="success"
                    variant="outlined"
                    sx={{ height: 18, fontSize: '0.65rem' }}
                  />
                ) : (
                  <Chip
                    icon={<VisibilityIcon sx={{ fontSize: 12 }} />}
                    label="You reviewed"
                    size="small"
                    variant="outlined"
                    sx={{ height: 18, fontSize: '0.65rem' }}
                  />
                )
              )}
              {!isMerged && pr.is_new && (
                <Chip label="NEW" size="small" color="success" sx={{ height: 18, fontSize: '0.65rem' }} />
              )}
            </Box>

            <Typography
              variant="body2"
              sx={{
                fontWeight: 500,
                lineHeight: 1.3,
                display: '-webkit-box',
                WebkitLineClamp: 2,
                WebkitBoxOrient: 'vertical',
                overflow: 'hidden',
                mb: 0.5,
              }}
            >
              {pr.title}
            </Typography>

            {hasStatusRow && (
              <Stack direction="row" spacing={0.5} sx={{ flexWrap: 'wrap', gap: 0.5, mb: 0.5 }}>
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

            <Stack direction="row" spacing={0.75} sx={{ alignItems: 'center' }}>
              <Tooltip title={pr.author} arrow>
                <Avatar
                  src={ghAvatar(pr.author)}
                  alt={pr.author}
                  sx={{ width: 24, height: 24, fontSize: '0.7rem' }}
                />
              </Tooltip>
              {reviewerUsernames.length > 0 && (
                <AvatarGroup
                  max={3}
                  sx={{
                    '& .MuiAvatar-root': { width: 20, height: 20, fontSize: '0.6rem', border: '1px solid', borderColor: 'background.paper' },
                  }}
                >
                  {reviewerUsernames.map((u) => (
                    <Tooltip key={u} title={u} arrow>
                      <Avatar src={ghAvatar(u, 40)} alt={u} />
                    </Tooltip>
                  ))}
                </AvatarGroup>
              )}
              <Typography variant="caption" color="text.secondary" noWrap sx={{ ml: 0.25 }}>
                {isMerged ? `merged ${formatDistanceToNow(new Date(pr.updated_at), { addSuffix: true })}` : formatDistanceToNow(new Date(pr.created_at), { addSuffix: true })}
              </Typography>
              <Typography variant="caption" color="text.secondary" noWrap>
                +{pr.additions} -{pr.deletions}
              </Typography>
            </Stack>
          </CardContent>
        </CardActionArea>
      </Card>
    </Tooltip>
  );
}
