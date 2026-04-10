import { useMemo, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import Box from '@mui/material/Box';
import Typography from '@mui/material/Typography';
import Chip from '@mui/material/Chip';
import Avatar from '@mui/material/Avatar';
import AvatarGroup from '@mui/material/AvatarGroup';
import Tooltip from '@mui/material/Tooltip';
import Divider from '@mui/material/Divider';
import ButtonBase from '@mui/material/ButtonBase';
import IconButton from '@mui/material/IconButton';
import ToggleButtonGroup from '@mui/material/ToggleButtonGroup';
import ToggleButton from '@mui/material/ToggleButton';
import MergeIcon from '@mui/icons-material/MergeType';
import CheckIcon from '@mui/icons-material/Check';
import VisibilityIcon from '@mui/icons-material/Visibility';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import ExpandLessIcon from '@mui/icons-material/ExpandLess';
import { formatDistanceToNow, subDays } from 'date-fns';
import { parseJSON } from '../../utils/parseJson.ts';
import { ghAvatar } from '../../utils/ghAvatar.ts';
import type { PRWithReview, HumanReviewSummary } from '../../types';

const COLLAPSED_COUNT = 10;

type DayFilter = 1 | 3 | 7;

function getReviewerUsernames(summary: HumanReviewSummary | null): string[] {
  if (!summary) return [];
  const set = new Set<string>();
  for (const u of summary.approved_by ?? []) set.add(u);
  for (const u of summary.changes_requested_by ?? []) set.add(u);
  for (const u of summary.commented_by ?? []) set.add(u);
  return [...set];
}

interface MergedRowProps {
  pr: PRWithReview;
}

function MergedRow({ pr }: MergedRowProps) {
  const navigate = useNavigate();
  const reviewSummary = useMemo(
    () => parseJSON<HumanReviewSummary>(pr.human_review_summary),
    [pr.human_review_summary],
  );
  const reviewerUsernames = useMemo(() => getReviewerUsernames(reviewSummary), [reviewSummary]);

  return (
    <ButtonBase
      onClick={() => navigate(`/prs/${pr.id}`)}
      sx={{
        display: 'flex',
        alignItems: 'center',
        gap: 1,
        px: 1.5,
        py: 0.75,
        width: '100%',
        textAlign: 'left',
        borderRadius: 0.5,
        '&:hover': { bgcolor: 'action.hover' },
      }}
    >
      <Tooltip title={pr.author} arrow>
        <Avatar
          src={ghAvatar(pr.author)}
          alt={pr.author}
          sx={{ width: 24, height: 24, fontSize: '0.7rem', flexShrink: 0 }}
        />
      </Tooltip>

      <Typography
        variant="caption"
        color="text.secondary"
        sx={{ fontWeight: 500, flexShrink: 0, minWidth: 0 }}
        noWrap
      >
        {pr.repo} #{pr.pr_number}
      </Typography>

      <Typography
        variant="body2"
        sx={{
          fontWeight: 500,
          fontSize: '0.8rem',
          flex: 1,
          minWidth: 0,
          overflow: 'hidden',
          textOverflow: 'ellipsis',
          whiteSpace: 'nowrap',
        }}
      >
        {pr.title}
      </Typography>

      <Chip
        icon={<MergeIcon sx={{ fontSize: 12 }} />}
        label="MERGED"
        size="small"
        color="info"
        sx={{ height: 18, fontSize: '0.6rem', flexShrink: 0 }}
      />

      {pr.my_review && (
        pr.my_review.review_state === 'approved' ? (
          <Chip
            icon={<CheckIcon sx={{ fontSize: 12 }} />}
            label="You approved"
            size="small"
            color="success"
            variant="outlined"
            sx={{ height: 18, fontSize: '0.6rem', flexShrink: 0 }}
          />
        ) : (
          <Chip
            icon={<VisibilityIcon sx={{ fontSize: 12 }} />}
            label="You reviewed"
            size="small"
            variant="outlined"
            sx={{ height: 18, fontSize: '0.6rem', flexShrink: 0 }}
          />
        )
      )}

      {reviewerUsernames.length > 0 && (
        <AvatarGroup
          max={3}
          sx={{
            flexShrink: 0,
            '& .MuiAvatar-root': { width: 18, height: 18, fontSize: '0.55rem', border: '1px solid', borderColor: 'background.paper' },
          }}
        >
          {reviewerUsernames.map((u) => (
            <Tooltip key={u} title={u} arrow>
              <Avatar src={ghAvatar(u, 32)} alt={u} />
            </Tooltip>
          ))}
        </AvatarGroup>
      )}

      <Typography variant="caption" color="text.secondary" sx={{ flexShrink: 0, whiteSpace: 'nowrap' }}>
        merged {formatDistanceToNow(new Date(pr.updated_at), { addSuffix: false })} ago
      </Typography>

      <Typography variant="caption" color="text.secondary" sx={{ flexShrink: 0, whiteSpace: 'nowrap' }}>
        +{pr.additions} &minus;{pr.deletions}
      </Typography>
    </ButtonBase>
  );
}

interface MergedListProps {
  prs: PRWithReview[];
}

export function MergedList({ prs }: MergedListProps) {
  const [expanded, setExpanded] = useState(false);
  const [dayFilter, setDayFilter] = useState<DayFilter>(7);

  const filteredPrs = useMemo(() => {
    const cutoff = subDays(new Date(), dayFilter);
    return prs.filter((pr) => new Date(pr.updated_at) >= cutoff);
  }, [prs, dayFilter]);

  if (prs.length === 0) return null;

  const canExpand = filteredPrs.length > COLLAPSED_COUNT;
  const visiblePrs = expanded ? filteredPrs : filteredPrs.slice(0, COLLAPSED_COUNT);

  return (
    <Box sx={{ flexShrink: 0, pb: 1 }}>
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 0.5, px: 1.5 }}>
        <MergeIcon sx={{ fontSize: 18, color: 'info.main' }} />
        <Typography variant="subtitle2" sx={{ fontWeight: 600 }}>
          Recently Merged
        </Typography>
        <Chip label={filteredPrs.length} size="small" sx={{ height: 20, fontSize: '0.7rem', fontWeight: 600 }} />

        <ToggleButtonGroup
          value={dayFilter}
          exclusive
          onChange={(_, v) => { if (v !== null) { setDayFilter(v); setExpanded(false); } }}
          size="small"
          sx={{ height: 22, ml: 0.5 }}
        >
          <ToggleButton value={1} sx={{ px: 0.75, py: 0, fontSize: '0.65rem', lineHeight: 1 }}>1d</ToggleButton>
          <ToggleButton value={3} sx={{ px: 0.75, py: 0, fontSize: '0.65rem', lineHeight: 1 }}>3d</ToggleButton>
          <ToggleButton value={7} sx={{ px: 0.75, py: 0, fontSize: '0.65rem', lineHeight: 1 }}>7d</ToggleButton>
        </ToggleButtonGroup>

        {canExpand && (
          <Tooltip title={expanded ? 'Show less' : `Show all ${filteredPrs.length}`} arrow>
            <IconButton size="small" onClick={() => setExpanded(!expanded)} sx={{ ml: 'auto' }}>
              {expanded ? <ExpandLessIcon fontSize="small" /> : <ExpandMoreIcon fontSize="small" />}
            </IconButton>
          </Tooltip>
        )}
      </Box>
      <Box
        sx={{
          bgcolor: 'background.paper',
          borderRadius: 1,
          border: 1,
          borderColor: 'divider',
          overflow: 'hidden',
          maxHeight: expanded ? 'none' : 380,
          overflowY: expanded ? 'visible' : 'auto',
        }}
      >
        {visiblePrs.length === 0 ? (
          <Box sx={{ py: 2, textAlign: 'center' }}>
            <Typography variant="body2" color="text.secondary">
              No PRs merged in the last {dayFilter === 1 ? 'day' : `${dayFilter} days`}
            </Typography>
          </Box>
        ) : (
          visiblePrs.map((pr, i) => (
            <Box key={pr.id}>
              {i > 0 && <Divider />}
              <MergedRow pr={pr} />
            </Box>
          ))
        )}
      </Box>
      {canExpand && !expanded && (
        <ButtonBase
          onClick={() => setExpanded(true)}
          sx={{
            width: '100%',
            py: 0.5,
            justifyContent: 'center',
            borderRadius: 1,
            mt: 0.5,
            '&:hover': { bgcolor: 'action.hover' },
          }}
        >
          <Typography variant="caption" color="primary" sx={{ fontWeight: 500 }}>
            Show all {filteredPrs.length} merged PRs
          </Typography>
        </ButtonBase>
      )}
    </Box>
  );
}
