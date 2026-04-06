import { useState, useEffect, useCallback, useMemo } from 'react';
import Container from '@mui/material/Container';
import AppBar from '@mui/material/AppBar';
import Toolbar from '@mui/material/Toolbar';
import Typography from '@mui/material/Typography';
import Button from '@mui/material/Button';
import Box from '@mui/material/Box';
import Snackbar from '@mui/material/Snackbar';
import Alert from '@mui/material/Alert';
import Tabs from '@mui/material/Tabs';
import Tab from '@mui/material/Tab';
import RefreshIcon from '@mui/icons-material/Refresh';
import RadarIcon from '@mui/icons-material/Radar';
import { DigestCards } from '../components/digest/DigestCards.tsx';
import { PRList } from '../components/pr/PRList.tsx';
import { PRFilters } from '../components/pr/PRFilters.tsx';
import { fetchDigest, fetchPRs, triggerScan } from '../services/api.ts';
import type { Digest, PRWithReview } from '../types';

export function DashboardPage() {
  const [digest, setDigest] = useState<Digest | null>(null);
  const [prs, setPrs] = useState<PRWithReview[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [scanning, setScanning] = useState(false);
  const [snackbar, setSnackbar] = useState<{ message: string; severity: 'success' | 'error' } | null>(null);
  const [tab, setTab] = useState(0);

  // Filters
  const [repoFilter, setRepoFilter] = useState('');
  const [reviewStatusFilter, setReviewStatusFilter] = useState('');
  const [ciStatusFilter, setCIStatusFilter] = useState('');
  const [coderabbitStatusFilter, setCodeRabbitStatusFilter] = useState('');
  const [newOnly, setNewOnly] = useState(false);
  const [repos, setRepos] = useState<string[]>([]);

  const myPrs = useMemo(() => prs.filter((p) => p.is_my_pr), [prs]);
  const displayedPrs = tab === 0 ? prs : myPrs;

  const loadData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [digestData, prData] = await Promise.all([
        fetchDigest(),
        fetchPRs({
          ...(repoFilter && { repo: repoFilter }),
          ...(reviewStatusFilter && { my_review_status: reviewStatusFilter }),
          ...(ciStatusFilter && { ci_status: ciStatusFilter }),
          ...(coderabbitStatusFilter && { coderabbit_status: coderabbitStatusFilter }),
          ...(newOnly && { is_new: 'true' }),
          per_page: 500,
        }),
      ]);
      setDigest(digestData);
      setPrs(prData.items || []);

      // Extract unique repos for filter dropdown
      const uniqueRepos = [...new Set((prData.items || []).map((p: PRWithReview) => p.repo))].sort();
      setRepos((prev) => {
        if (uniqueRepos.length > prev.length) return uniqueRepos;
        return prev;
      });
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load data');
    } finally {
      setLoading(false);
    }
  }, [repoFilter, reviewStatusFilter, ciStatusFilter, coderabbitStatusFilter, newOnly]);

  useEffect(() => {
    loadData();
  }, [loadData]);

  const handleScan = async () => {
    setScanning(true);
    try {
      const scan = await triggerScan();
      setSnackbar({
        message: `Scan completed: ${scan.prs_found} PRs found (${scan.new_prs} new)`,
        severity: 'success',
      });
      loadData();
    } catch (err) {
      setSnackbar({
        message: err instanceof Error ? err.message : 'Scan failed',
        severity: 'error',
      });
    } finally {
      setScanning(false);
    }
  };

  return (
    <>
      <AppBar position="static" color="default" elevation={1}>
        <Toolbar>
          <RadarIcon sx={{ mr: 1 }} />
          <Typography variant="h6" sx={{ flexGrow: 1 }}>
            PR Scout
          </Typography>
          <Button
            variant="outlined"
            startIcon={<RefreshIcon />}
            onClick={handleScan}
            disabled={scanning}
            size="small"
          >
            {scanning ? 'Scanning...' : 'Run Scan'}
          </Button>
        </Toolbar>
      </AppBar>

      <Container maxWidth="lg" sx={{ mt: 2 }}>
        <DigestCards digest={digest} />

        <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 1 }}>
          <Tabs value={tab} onChange={(_, v) => setTab(v)} sx={{ minHeight: 36 }}>
            <Tab label={`All PRs (${prs.length})`} sx={{ minHeight: 36, py: 0 }} />
            <Tab label={`My PRs (${myPrs.length})`} sx={{ minHeight: 36, py: 0 }} />
          </Tabs>
          <Typography variant="body2" color="text.secondary">
            {displayedPrs.length} shown
          </Typography>
        </Box>

        <PRFilters
          repo={repoFilter}
          onRepoChange={setRepoFilter}
          reviewStatus={reviewStatusFilter}
          onReviewStatusChange={setReviewStatusFilter}
          ciStatus={ciStatusFilter}
          onCIStatusChange={setCIStatusFilter}
          coderabbitStatus={coderabbitStatusFilter}
          onCodeRabbitStatusChange={setCodeRabbitStatusFilter}
          newOnly={newOnly}
          onNewOnlyChange={setNewOnly}
          repos={repos}
        />

        <PRList prs={displayedPrs} loading={loading} error={error} />
      </Container>

      <Snackbar
        open={!!snackbar}
        autoHideDuration={5000}
        onClose={() => setSnackbar(null)}
        anchorOrigin={{ vertical: 'bottom', horizontal: 'center' }}
      >
        {snackbar ? (
          <Alert severity={snackbar.severity} onClose={() => setSnackbar(null)} variant="filled">
            {snackbar.message}
          </Alert>
        ) : undefined}
      </Snackbar>
    </>
  );
}
