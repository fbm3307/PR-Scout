import Box from '@mui/material/Box';
import Typography from '@mui/material/Typography';
import { StatCard } from './StatCard.tsx';
import type { Digest } from '../../types';

interface DigestCardsProps {
  digest: Digest | null;
}

export function DigestCards({ digest }: DigestCardsProps) {
  if (!digest) return null;

  return (
    <Box sx={{ mb: 3 }}>
      {digest.scan && (
        <Typography variant="body2" color="text.secondary" sx={{ mb: 1.5 }}>
          Last scan: {new Date(digest.scan.started_at).toLocaleString()} — {digest.scan.repos_scanned} repos scanned
        </Typography>
      )}
      <Box sx={{ display: 'flex', gap: 2, flexWrap: 'wrap' }}>
        <StatCard title="Open PRs" value={digest.total_open_prs} />
        <StatCard title="New PRs" value={digest.new_prs} color="success.main" />
        <StatCard title="Need Attention" value={digest.needs_attention} color="warning.main" />
        <StatCard title="Active Repos" value={digest.repos_with_activity} color="info.main" />
      </Box>
    </Box>
  );
}
