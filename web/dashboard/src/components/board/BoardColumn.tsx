import Box from '@mui/material/Box';
import Typography from '@mui/material/Typography';
import Chip from '@mui/material/Chip';
import Stack from '@mui/material/Stack';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import { BoardCard } from './BoardCard.tsx';
import type { BoardColumnData } from '../../hooks/useBoardColumns.ts';

interface BoardColumnProps {
  column: BoardColumnData;
}

export function BoardColumn({ column }: BoardColumnProps) {
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
            label={column.prs.length}
            size="small"
            sx={{ height: 20, fontSize: '0.7rem', fontWeight: 600 }}
          />
          {column.mergeReadyCount > 0 && (
            <Chip
              icon={<CheckCircleIcon sx={{ fontSize: 12 }} />}
              label={`${column.mergeReadyCount} ready`}
              size="small"
              color="success"
              variant="outlined"
              sx={{ height: 20, fontSize: '0.65rem' }}
            />
          )}
        </Stack>
      </Box>

      <Box sx={{ flex: 1, overflowY: 'auto', p: 1 }}>
        {column.prs.length === 0 ? (
          <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '100%', minHeight: 80 }}>
            <Typography variant="body2" color="text.secondary">
              No PRs
            </Typography>
          </Box>
        ) : (
          column.prs.map((pr) => <BoardCard key={pr.id} pr={pr} />)
        )}
      </Box>
    </Box>
  );
}
