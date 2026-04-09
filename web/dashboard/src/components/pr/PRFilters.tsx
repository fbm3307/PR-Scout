import Box from '@mui/material/Box';
import TextField from '@mui/material/TextField';
import MenuItem from '@mui/material/MenuItem';
import FormControlLabel from '@mui/material/FormControlLabel';
import Switch from '@mui/material/Switch';

interface PRFiltersProps {
  repo: string;
  onRepoChange: (value: string) => void;
  reviewStatus: string;
  onReviewStatusChange: (value: string) => void;
  ciStatus: string;
  onCIStatusChange: (value: string) => void;
  coderabbitStatus: string;
  onCodeRabbitStatusChange: (value: string) => void;
  newOnly: boolean;
  onNewOnlyChange: (value: boolean) => void;
  repos: string[];
  viewMode?: 'list' | 'board';
}

export function PRFilters({
  repo, onRepoChange,
  reviewStatus, onReviewStatusChange,
  ciStatus, onCIStatusChange,
  coderabbitStatus, onCodeRabbitStatusChange,
  newOnly, onNewOnlyChange,
  repos,
  viewMode,
}: PRFiltersProps) {
  return (
    <Box sx={{ display: 'flex', gap: 2, mb: 2, flexWrap: 'wrap', alignItems: 'center' }}>
      <TextField
        select
        label="Repository"
        value={repo}
        onChange={(e) => onRepoChange(e.target.value)}
        size="small"
        sx={{ minWidth: 180 }}
      >
        <MenuItem value="">All Repos</MenuItem>
        {repos.map((r) => (
          <MenuItem key={r} value={r}>{r}</MenuItem>
        ))}
      </TextField>

      {viewMode !== 'board' && (
        <TextField
          select
          label="My Review Status"
          value={reviewStatus}
          onChange={(e) => onReviewStatusChange(e.target.value)}
          size="small"
          sx={{ minWidth: 180 }}
        >
          <MenuItem value="">All</MenuItem>
          <MenuItem value="needs_attention">Needs Attention</MenuItem>
          <MenuItem value="waiting">Waiting</MenuItem>
          <MenuItem value="approved">Approved</MenuItem>
          <MenuItem value="not_reviewed">Not Reviewed</MenuItem>
        </TextField>
      )}

      <TextField
        select
        label="CI Status"
        value={ciStatus}
        onChange={(e) => onCIStatusChange(e.target.value)}
        size="small"
        sx={{ minWidth: 150 }}
      >
        <MenuItem value="">All</MenuItem>
        <MenuItem value="success">Passing</MenuItem>
        <MenuItem value="failure">Failing</MenuItem>
        <MenuItem value="pending">Pending</MenuItem>
      </TextField>

      <TextField
        select
        label="CodeRabbit"
        value={coderabbitStatus}
        onChange={(e) => onCodeRabbitStatusChange(e.target.value)}
        size="small"
        sx={{ minWidth: 160 }}
      >
        <MenuItem value="">All</MenuItem>
        <MenuItem value="reviewed">Reviewed by CodeRabbit</MenuItem>
        <MenuItem value="all_resolved">All Resolved</MenuItem>
        <MenuItem value="has_unresolved">Has Unresolved</MenuItem>
        <MenuItem value="no_review">No CodeRabbit</MenuItem>
      </TextField>

      <FormControlLabel
        control={<Switch checked={newOnly} onChange={(e) => onNewOnlyChange(e.target.checked)} size="small" />}
        label="New PRs only"
      />
    </Box>
  );
}
