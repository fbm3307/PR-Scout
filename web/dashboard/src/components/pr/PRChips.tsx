import Chip from '@mui/material/Chip';
import Tooltip from '@mui/material/Tooltip';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import CancelIcon from '@mui/icons-material/Cancel';
import AccessTimeIcon from '@mui/icons-material/AccessTime';
import PersonIcon from '@mui/icons-material/Person';
import ShieldIcon from '@mui/icons-material/Shield';
import SmartToyIcon from '@mui/icons-material/SmartToy';
import type { HumanReviewSummary, CIStatus } from '../../types';

export function HumanReviewChip({ summary }: { summary: HumanReviewSummary }) {
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

export function CIStatusChip({ ci }: { ci: CIStatus }) {
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

export function RequiredChecksChip({ ci }: { ci: CIStatus }) {
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

export function CodeRabbitChip({ total, resolved }: { total: number; resolved: number }) {
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

export function CodeRabbitSummaryChip({ summary }: { summary: string }) {
  return (
    <Tooltip title={summary} arrow>
      <Chip
        icon={<SmartToyIcon sx={{ fontSize: 14 }} />}
        label="CodeRabbit"
        size="small"
        variant="outlined"
        sx={{ height: 22, fontSize: '0.7rem' }}
      />
    </Tooltip>
  );
}
