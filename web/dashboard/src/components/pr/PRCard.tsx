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
import EditNoteIcon from '@mui/icons-material/EditNote';
import HistoryIcon from '@mui/icons-material/History';
import { formatDistanceToNow, subDays } from 'date-fns';
import { parseJSON } from '../../utils/parseJson.ts';
import { ghAvatar } from '../../utils/ghAvatar.ts';
import { isMergeReady } from '../../utils/mergeReadiness.ts';
import { HumanReviewChip, CIStatusChip, RequiredChecksChip, CodeRabbitChip, CodeRabbitSummaryChip } from './PRChips.tsx';
import { ReviewStatusBadge } from '../review/ReviewStatusBadge.tsx';
import type { PRWithReview, HumanReviewSummary, CIStatus } from '../../types';

function getReviewerUsernames(summary: HumanReviewSummary | null): string[] {
  if (!summary) return [];
  const set = new Set<string>();
  for (const u of summary.approved_by ?? []) set.add(u);
  for (const u of summary.changes_requested_by ?? []) set.add(u);
  for (const u of summary.commented_by ?? []) set.add(u);
  return [...set];
}

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
  const reviewerUsernames = useMemo(() => getReviewerUsernames(reviewSummary), [reviewSummary]);

  const isStale = new Date(pr.updated_at) < subDays(new Date(), 90);
  const mergeReady = isMergeReady(pr);

  const hasCodeRabbit = pr.coderabbit_total > 0 || !!pr.coderabbit_summary;
  const hasStatusRow = reviewSummary?.total_reviewers || ciStatus?.total_checks || hasCodeRabbit;

  const borderColor = pr.is_draft ? 'text.disabled' : isStale ? 'warning.main' : mergeReady ? 'success.main' : undefined;

  return (
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
        <CardContent sx={{ py: 2, px: 2, '&:last-child': { pb: 2 } }}>
          <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 1.5 }}>
            <Box sx={{ flex: 1, minWidth: 0 }}>
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.75, mb: 0.5 }}>
                <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 500 }}>
                  {pr.repo}
                </Typography>
                <Typography variant="body2" color="text.secondary">
                  #{pr.pr_number}
                </Typography>
                {pr.is_draft && (
                  <Chip
                    icon={<EditNoteIcon sx={{ fontSize: 14 }} />}
                    label="DRAFT"
                    size="small"
                    sx={{ height: 20, fontSize: '0.7rem', bgcolor: 'action.selected' }}
                  />
                )}
                {isStale && (
                  <Chip
                    icon={<HistoryIcon sx={{ fontSize: 14 }} />}
                    label="STALE"
                    size="small"
                    color="warning"
                    variant="outlined"
                    sx={{ height: 20, fontSize: '0.7rem' }}
                  />
                )}
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

              <Stack direction="row" spacing={1} sx={{ mt: 1, alignItems: 'center' }}>
                <Tooltip title={pr.author} arrow>
                  <Avatar
                    src={ghAvatar(pr.author)}
                    alt={pr.author}
                    sx={{ width: 28, height: 28, fontSize: '0.75rem' }}
                  />
                </Tooltip>
                <Typography variant="caption" color="text.secondary" sx={{ fontWeight: 500 }}>
                  {pr.author}
                </Typography>
                {reviewerUsernames.length > 0 && (
                  <AvatarGroup
                    max={3}
                    sx={{
                      '& .MuiAvatar-root': { width: 22, height: 22, fontSize: '0.6rem', border: '1px solid', borderColor: 'background.paper' },
                    }}
                  >
                    {reviewerUsernames.map((u) => (
                      <Tooltip key={u} title={u} arrow>
                        <Avatar src={ghAvatar(u, 44)} alt={u} />
                      </Tooltip>
                    ))}
                  </AvatarGroup>
                )}
                <Typography variant="caption" color="text.secondary">
                  {formatDistanceToNow(new Date(pr.created_at), { addSuffix: true })}
                </Typography>
                <Typography variant="caption" color="text.secondary" sx={{ fontWeight: 600 }}>
                  +{pr.additions} -{pr.deletions}
                </Typography>
              </Stack>
            </Box>

            <Box sx={{ flexShrink: 0 }}>
              <ReviewStatusBadge review={pr.my_review} isMyPR={pr.is_my_pr} />
            </Box>
          </Box>
        </CardContent>
      </CardActionArea>
    </Card>
  );
}
