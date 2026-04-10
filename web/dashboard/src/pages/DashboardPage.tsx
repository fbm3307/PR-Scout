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
import ToggleButtonGroup from '@mui/material/ToggleButtonGroup';
import ToggleButton from '@mui/material/ToggleButton';
import Tooltip from '@mui/material/Tooltip';
import RefreshIcon from '@mui/icons-material/Refresh';
import RadarIcon from '@mui/icons-material/Radar';
import ViewListIcon from '@mui/icons-material/ViewList';
import DashboardIcon from '@mui/icons-material/Dashboard';
import { DigestCards } from '../components/digest/DigestCards.tsx';
import { PRList } from '../components/pr/PRList.tsx';
import { PRFilters } from '../components/pr/PRFilters.tsx';
import { BoardView } from '../components/board/BoardView.tsx';
import { useBoardColumns } from '../hooks/useBoardColumns.ts';
import { fetchDigest, fetchPRs, triggerScan } from '../services/api.ts';
import type { Digest, PRWithReview } from '../types';

type ViewMode = 'list' | 'board';

function getInitialViewMode(): ViewMode {
  const stored = localStorage.getItem('pr-scout-view-mode');
  return stored === 'board' ? 'board' : 'list';
}

export function DashboardPage() {
  const [digest, setDigest] = useState<Digest | null>(null);
  const [prs, setPrs] = useState<PRWithReview[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [scanning, setScanning] = useState(false);
  const [snackbar, setSnackbar] = useState<{ message: string; severity: 'success' | 'error' } | null>(null);
  const [tab, setTab] = useState(0);
  const [viewMode, setViewMode] = useState<ViewMode>(getInitialViewMode);

  // Filters
  const [repoFilter, setRepoFilter] = useState('');
  const [reviewStatusFilter, setReviewStatusFilter] = useState('');
  const [ciStatusFilter, setCIStatusFilter] = useState('');
  const [coderabbitStatusFilter, setCodeRabbitStatusFilter] = useState('');
  const [newOnly, setNewOnly] = useState(false);
  const [repos, setRepos] = useState<string[]>([]);

  const isBoardMode = viewMode === 'board' && tab === 0;

  const openPrs = useMemo(() => prs.filter((p) => p.state !== 'merged'), [prs]);
  const myPrs = useMemo(() => openPrs.filter((p) => p.is_my_pr), [openPrs]);
  const reviewPrs = useMemo(() => prs.filter((p) => !p.is_my_pr), [prs]);
  const displayedPrs = tab === 0 ? openPrs : myPrs;

  const columns = useBoardColumns(isBoardMode ? reviewPrs : []);

  const handleViewModeChange = (_: unknown, newMode: ViewMode | null) => {
    if (!newMode) return;
    setViewMode(newMode);
    localStorage.setItem('pr-scout-view-mode', newMode);
    if (newMode === 'board') {
      setReviewStatusFilter('');
    }
  };

  const loadData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [digestData, prData] = await Promise.all([
        fetchDigest(),
        fetchPRs({
          ...(repoFilter && { repo: repoFilter }),
          ...(!isBoardMode && reviewStatusFilter && { my_review_status: reviewStatusFilter }),
          ...(ciStatusFilter && { ci_status: ciStatusFilter }),
          ...(coderabbitStatusFilter && { coderabbit_status: coderabbitStatusFilter }),
          ...(newOnly && { is_new: 'true' }),
          per_page: 500,
        }),
      ]);
      setDigest(digestData);
      setPrs(prData.items || []);

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
  }, [repoFilter, reviewStatusFilter, ciStatusFilter, coderabbitStatusFilter, newOnly, isBoardMode]);

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

  const shownCount = isBoardMode ? reviewPrs.length : displayedPrs.length;

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', height: '100dvh' }}>
      <AppBar position="static" color="default" elevation={isBoardMode ? 0 : 1} sx={isBoardMode ? { borderBottom: 1, borderColor: 'divider' } : undefined}>
        <Toolbar variant={isBoardMode ? 'dense' : 'regular'}>
          <RadarIcon sx={{ mr: 1 }} />
          <Typography variant="h6" sx={{ flexGrow: 1 }}>
            PR Scout
          </Typography>

          {isBoardMode && (
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5, mr: 2 }}>
              <Tabs value={tab} onChange={(_, v) => setTab(v)} sx={{ minHeight: 36 }}>
                <Tab
                  label={`Review Board (${reviewPrs.length})`}
                  sx={{ minHeight: 36, py: 0 }}
                />
                <Tab label={`My PRs (${myPrs.length})`} sx={{ minHeight: 36, py: 0 }} />
              </Tabs>
              <ToggleButtonGroup
                value={viewMode}
                exclusive
                onChange={handleViewModeChange}
                size="small"
                sx={{ height: 28 }}
              >
                <ToggleButton value="list" sx={{ px: 0.75 }}>
                  <ViewListIcon sx={{ fontSize: 16 }} />
                </ToggleButton>
                <ToggleButton value="board" sx={{ px: 0.75 }}>
                  <DashboardIcon sx={{ fontSize: 16 }} />
                </ToggleButton>
              </ToggleButtonGroup>
              <Typography variant="body2" color="text.secondary">
                {shownCount} shown
              </Typography>
            </Box>
          )}

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

      <Container
        maxWidth={isBoardMode ? false : 'lg'}
        sx={{
          mt: isBoardMode ? 0 : 2,
          px: isBoardMode ? 2 : undefined,
          flex: 1,
          minHeight: 0,
          overflow: isBoardMode ? 'hidden' : 'auto',
          display: 'flex',
          flexDirection: 'column',
        }}
      >
        {!isBoardMode && <DigestCards digest={digest} />}

        {!isBoardMode && (
          <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 1 }}>
            <Tabs value={tab} onChange={(_, v) => setTab(v)} sx={{ minHeight: 36 }}>
              <Tab
                label={`All PRs (${openPrs.length})`}
                sx={{ minHeight: 36, py: 0 }}
              />
              <Tab label={`My PRs (${myPrs.length})`} sx={{ minHeight: 36, py: 0 }} />
            </Tabs>

            <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
              <ToggleButtonGroup
                value={viewMode}
                exclusive
                onChange={handleViewModeChange}
                size="small"
                sx={{ height: 30 }}
              >
                <ToggleButton value="list" sx={{ px: 1 }}>
                  <ViewListIcon sx={{ fontSize: 18 }} />
                </ToggleButton>
                <Tooltip title={tab !== 0 ? 'Board view available for All PRs' : ''}>
                  <span>
                    <ToggleButton value="board" sx={{ px: 1 }} disabled={tab !== 0}>
                      <DashboardIcon sx={{ fontSize: 18 }} />
                    </ToggleButton>
                  </span>
                </Tooltip>
              </ToggleButtonGroup>
              <Typography variant="body2" color="text.secondary">
                {shownCount} shown
              </Typography>
            </Box>
          </Box>
        )}

        {!isBoardMode && (
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
            viewMode="list"
          />
        )}

        {isBoardMode ? (
          <BoardView columns={columns} loading={loading} />
        ) : (
          <PRList prs={displayedPrs} loading={loading} error={error} />
        )}
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
    </Box>
  );
}
