import Box from '@mui/material/Box';
import Typography from '@mui/material/Typography';
import CircularProgress from '@mui/material/CircularProgress';
import { PRCard } from './PRCard.tsx';
import type { PRWithReview } from '../../types';

interface PRListProps {
  prs: PRWithReview[];
  loading: boolean;
  error: string | null;
}

export function PRList({ prs, loading, error }: PRListProps) {
  if (loading) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', py: 4 }}>
        <CircularProgress />
      </Box>
    );
  }

  if (error) {
    return (
      <Typography color="error" sx={{ py: 2 }}>
        Error loading PRs: {error}
      </Typography>
    );
  }

  if (prs.length === 0) {
    return (
      <Typography color="text.secondary" sx={{ py: 4, textAlign: 'center' }}>
        No PRs found. Run a scan to get started.
      </Typography>
    );
  }

  return (
    <Box>
      {prs.map((pr) => (
        <PRCard key={pr.id} pr={pr} />
      ))}
    </Box>
  );
}
