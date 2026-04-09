import Box from '@mui/material/Box';
import Skeleton from '@mui/material/Skeleton';
import Typography from '@mui/material/Typography';
import { BoardColumn } from './BoardColumn.tsx';
import type { BoardColumnData } from '../../hooks/useBoardColumns.ts';

interface BoardViewProps {
  columns: BoardColumnData[];
  loading?: boolean;
}

const SKELETON_COLUMN_LABELS = ['Not Reviewed', 'Needs Attention', 'Waiting', 'Approved', 'Recently Merged'];

function BoardSkeleton() {
  return (
    <Box sx={{ display: 'flex', gap: 1.5, flex: 1, minHeight: 0 }}>
      {SKELETON_COLUMN_LABELS.map((label) => (
        <Box
          key={label}
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
            <Typography variant="subtitle2" sx={{ fontWeight: 600 }}>
              {label}
            </Typography>
          </Box>
          <Box sx={{ p: 1 }}>
            {[0, 1, 2].map((i) => (
              <Skeleton
                key={i}
                variant="rounded"
                height={120}
                sx={{ mb: 1, borderRadius: 1 }}
              />
            ))}
          </Box>
        </Box>
      ))}
    </Box>
  );
}

export function BoardView({ columns, loading }: BoardViewProps) {
  if (loading) {
    return <BoardSkeleton />;
  }

  return (
    <Box sx={{ display: 'flex', gap: 1.5, flex: 1, minHeight: 0 }}>
      {columns.map((col) => (
        <BoardColumn key={col.id} column={col} />
      ))}
    </Box>
  );
}
