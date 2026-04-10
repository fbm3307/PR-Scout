import { useState, useMemo } from 'react';
import Box from '@mui/material/Box';
import Typography from '@mui/material/Typography';
import Chip from '@mui/material/Chip';
import Stack from '@mui/material/Stack';
import ButtonBase from '@mui/material/ButtonBase';
import Divider from '@mui/material/Divider';
import ToggleButtonGroup from '@mui/material/ToggleButtonGroup';
import ToggleButton from '@mui/material/ToggleButton';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import ExpandLessIcon from '@mui/icons-material/ExpandLess';
import { subDays } from 'date-fns';
import { BoardCard } from './BoardCard.tsx';
import type { BoardColumnData } from '../../hooks/useBoardColumns.ts';

const STALE_DAYS = 90;

interface BoardColumnProps {
  column: BoardColumnData;
}

type DayFilter = 1 | 3 | 7;

export function BoardColumn({ column }: BoardColumnProps) {
  const isMergedColumn = column.id === 'recently_merged';
  const [dayFilter, setDayFilter] = useState<DayFilter>(7);
  const [staleExpanded, setStaleExpanded] = useState(false);

  const displayedPrs = useMemo(() => {
    if (!isMergedColumn || dayFilter === 7) return column.prs;
    const cutoff = subDays(new Date(), dayFilter);
    return column.prs.filter((pr) => new Date(pr.updated_at) >= cutoff);
  }, [column.prs, dayFilter, isMergedColumn]);

  const { activePrs, stalePrs } = useMemo(() => {
    if (isMergedColumn) return { activePrs: displayedPrs, stalePrs: [] };
    const cutoff = subDays(new Date(), STALE_DAYS);
    const active = [];
    const stale = [];
    for (const pr of displayedPrs) {
      if (new Date(pr.updated_at) < cutoff) {
        stale.push(pr);
      } else {
        active.push(pr);
      }
    }
    return { activePrs: active, stalePrs: stale };
  }, [displayedPrs, isMergedColumn]);

  const isEmpty = activePrs.length === 0 && stalePrs.length === 0;

  return (
    <Box
      sx={{
        flex: 1,
        minWidth: 0,
        display: 'flex',
        flexDirection: 'column',
        bgcolor: 'action.hover',
        borderRadius: 1,
        overflow: 'hidden',
      }}
    >
      <Box sx={{ px: 1.5, py: 1, borderBottom: 1, borderColor: 'divider', bgcolor: 'background.paper' }}>
        <Stack direction="row" spacing={0.75} sx={{ alignItems: 'center', flexWrap: 'wrap', gap: 0.5 }}>
          <Typography variant="subtitle2" sx={{ fontWeight: 600 }}>
            {column.label}
          </Typography>
          <Chip
            label={activePrs.length}
            size="small"
            sx={{ height: 20, fontSize: '0.7rem', fontWeight: 600 }}
          />
          {stalePrs.length > 0 && (
            <Typography variant="caption" color="text.disabled" sx={{ fontSize: '0.65rem' }}>
              +{stalePrs.length} stale
            </Typography>
          )}
          {!isMergedColumn && column.mergeReadyCount > 0 && (
            <Chip
              icon={<CheckCircleIcon sx={{ fontSize: 12 }} />}
              label={`${column.mergeReadyCount} ready`}
              size="small"
              color="success"
              variant="outlined"
              sx={{ height: 20, fontSize: '0.65rem' }}
            />
          )}
          {isMergedColumn && (
            <ToggleButtonGroup
              value={dayFilter}
              exclusive
              onChange={(_, v) => { if (v !== null) setDayFilter(v); }}
              size="small"
              sx={{ height: 20, ml: 0.5 }}
            >
              <ToggleButton value={1} sx={{ px: 0.5, py: 0, fontSize: '0.6rem', lineHeight: 1, minWidth: 0 }}>1d</ToggleButton>
              <ToggleButton value={3} sx={{ px: 0.5, py: 0, fontSize: '0.6rem', lineHeight: 1, minWidth: 0 }}>3d</ToggleButton>
              <ToggleButton value={7} sx={{ px: 0.5, py: 0, fontSize: '0.6rem', lineHeight: 1, minWidth: 0 }}>7d</ToggleButton>
            </ToggleButtonGroup>
          )}
        </Stack>
      </Box>

      <Box sx={{ flex: 1, overflowY: 'auto', p: 1 }}>
        {isEmpty ? (
          <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '100%', minHeight: 80 }}>
            <Typography variant="body2" color="text.secondary">
              No PRs
            </Typography>
          </Box>
        ) : (
          <>
            {activePrs.map((pr) => <BoardCard key={pr.id} pr={pr} />)}

            {stalePrs.length > 0 && (
              <>
                <Divider sx={{ my: 0.5 }} />
                <ButtonBase
                  onClick={() => setStaleExpanded(!staleExpanded)}
                  sx={{
                    width: '100%',
                    py: 0.5,
                    px: 1,
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    gap: 0.5,
                    borderRadius: 0.5,
                    '&:hover': { bgcolor: 'action.hover' },
                  }}
                >
                  {staleExpanded ? <ExpandLessIcon sx={{ fontSize: 16 }} /> : <ExpandMoreIcon sx={{ fontSize: 16 }} />}
                  <Typography variant="caption" color="text.secondary" sx={{ fontWeight: 500 }}>
                    {staleExpanded ? 'Hide' : 'Show'} {stalePrs.length} stale
                  </Typography>
                </ButtonBase>
                {staleExpanded && stalePrs.map((pr) => <BoardCard key={pr.id} pr={pr} />)}
              </>
            )}
          </>
        )}
      </Box>
    </Box>
  );
}
