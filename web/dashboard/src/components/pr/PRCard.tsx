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
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import CancelIcon from '@mui/icons-material/Cancel';
import AccessTimeIcon from '@mui/icons-material/AccessTime';
import PersonIcon from '@mui/icons-material/Person';
import ShieldIcon from '@mui/icons-material/Shield';
import SmartToyIcon from '@mui/icons-material/SmartToy';
import { formatDistanceToNow } from 'date-fns';
import { ReviewStatusBadge } from '../review/ReviewStatusBadge.tsx';
import type { PRWithReview, HumanReviewSummary, CIStatus } from '../../types';

interface PRCardProps {
  pr: PRWithReview;
}

function parseJSON<T>(raw: string | undefined): T | null {
  if (!raw) return null;
  try {
    return JSON.parse(raw) as T;
  } catch {
    return null;
  }
}

function HumanReviewChip({ summary }: { summary: HumanReviewSummary }) {
  if (summary.total_reviewers === 0) return null;

  const approved = summary.approved_by?.length ?? 0;
  const changesRequested = summary.changes_requested_by?.length ?? 0;
  const commented = summary.commented_by?.length ?? 0;

  let color: 'success' | 'warning' | 'default' = 'default';
  let label = `${summary.total_reviewers} reviewed`;
  if (approved > 0 && changesRequested === 0) {
    color = 'success';
    label = `${approved} approved`;
  } else if (changesRequested > 0) {
    color = 'warning';
    label = `${changesRequested} changes req.`;
  }

  const lines: string[] = [];
  if (approved > 0) lines.push(`Approved: ${summary.approved_by.join(', ')}`);
  if (changesRequested > 0) lines.push(`Changes requested: ${summary.changes_requested_by.join(', ')}`);
  if (commented > 0) lines.push(`Commented: ${summary.commented_by.join(', ')}`);

  return (
    <Tooltip title={lines.join('\n')} arrow>
      <Chip
        icon={<PersonIcon sx={{ fontSize: 14 }} />}
        label={label}
        size="small"
        color={color}
        variant="outlined"
        sx={{ height: 22, fontSize: '0.7rem' }}
      />
    </Tooltip>
  );
}

function CIStatusChip({ ci }: { ci: CIStatus }) {
  if (ci.total_checks === 0 && ci.overall_status === 'pending') return null;

  const iconMap = {
    success: <CheckCircleIcon sx={{ fontSize: 14 }} />,
    failure: <CancelIcon sx={{ fontSize: 14 }} />,
    pending: <AccessTimeIcon sx={{ fontSize: 14 }} />,
    mixed: <AccessTimeIcon sx={{ fontSize: 14 }} />,
  };
  const colorMap: Record<string, 'success' | 'error' | 'warning' | 'default'> = {
    success: 'success',
    failure: 'error',
    pending: 'warning',
    mixed: 'warning',
  };

  const label = `CI ${ci.passed}/${ci.total_checks}`;
  const tooltipLines = [`${ci.passed} passed, ${ci.failed} failed, ${ci.pending} pending`];
  if (ci.failed_checks?.length) {
    tooltipLines.push('');
    tooltipLines.push('Failed:');
    for (const f of ci.failed_checks) {
      const line = f.summary ? `${f.name}: ${f.summary}` : f.name;
      tooltipLines.push(`  ${line}`);
    }
  }

  return (
    <Tooltip title={tooltipLines.join('\n')} arrow>
      <Chip
        icon={iconMap[ci.overall_status]}
        label={label}
        size="small"
        color={colorMap[ci.overall_status] ?? 'default'}
        variant="outlined"
        sx={{ height: 22, fontSize: '0.7rem' }}
      />
    </Tooltip>
  );
}

function RequiredChecksChip({ ci }: { ci: CIStatus }) {
  if (ci.required_total === 0) return null;

  const allGreen = ci.required_all_green;
  return (
    <Tooltip title={`Required checks: ${ci.required_passed}/${ci.required_total} passing`} arrow>
      <Chip
        icon={<ShieldIcon sx={{ fontSize: 14 }} />}
        label={allGreen ? 'Required OK' : `Required ${ci.required_passed}/${ci.required_total}`}
        size="small"
        color={allGreen ? 'success' : 'error'}
        variant="outlined"
        sx={{ height: 22, fontSize: '0.7rem' }}
      />
    </Tooltip>
  );
}

function CodeRabbitChip({ total, resolved }: { total: number; resolved: number }) {
  if (total === 0) return null;

  let color: 'success' | 'warning' | 'error' = 'error';
  if (resolved === total) color = 'success';
  else if (resolved > 0) color = 'warning';

  return (
    <Tooltip title={`${resolved} of ${total} CodeRabbit comments resolved`} arrow>
      <Chip
        icon={<SmartToyIcon sx={{ fontSize: 14 }} />}
        label={`CodeRabbit ${resolved}/${total}`}
        size="small"
        color={color}
        variant="outlined"
        sx={{ height: 22, fontSize: '0.7rem' }}
      />
    </Tooltip>
  );
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
                    <Tooltip title={pr.coderabbit_summary} arrow>
                      <Chip
                        icon={<SmartToyIcon sx={{ fontSize: 14 }} />}
                        label="CodeRabbit"
                        size="small"
                        variant="outlined"
                        sx={{ height: 22, fontSize: '0.7rem' }}
                      />
                    </Tooltip>
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
