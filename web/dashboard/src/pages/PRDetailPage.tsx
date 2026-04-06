import { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import Container from '@mui/material/Container';
import AppBar from '@mui/material/AppBar';
import Toolbar from '@mui/material/Toolbar';
import Typography from '@mui/material/Typography';
import IconButton from '@mui/material/IconButton';
import Card from '@mui/material/Card';
import CardContent from '@mui/material/CardContent';
import Box from '@mui/material/Box';
import Chip from '@mui/material/Chip';
import Divider from '@mui/material/Divider';
import Link from '@mui/material/Link';
import CircularProgress from '@mui/material/CircularProgress';
import Stack from '@mui/material/Stack';
import ArrowBackIcon from '@mui/icons-material/ArrowBack';
import OpenInNewIcon from '@mui/icons-material/OpenInNew';
import ReactMarkdown from 'react-markdown';
import { formatDistanceToNow } from 'date-fns';
import { ReviewStatusBadge } from '../components/review/ReviewStatusBadge.tsx';
import { fetchPR } from '../services/api.ts';
import type { PRWithReview, ReviewComment } from '../types';

export function PRDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [pr, setPr] = useState<PRWithReview | null>(null);
  const [comments, setComments] = useState<ReviewComment[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!id) return;
    setLoading(true);
    fetchPR(Number(id))
      .then((data) => {
        setPr(data.pr);
        setComments(data.comments || []);
      })
      .catch((err) => setError(err instanceof Error ? err.message : 'Failed to load PR'))
      .finally(() => setLoading(false));
  }, [id]);

  if (loading) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', pt: 8 }}>
        <CircularProgress />
      </Box>
    );
  }

  if (error || !pr) {
    return (
      <Container maxWidth="md" sx={{ pt: 4 }}>
        <Typography color="error">{error || 'PR not found'}</Typography>
      </Container>
    );
  }

  const botComments = comments.filter((c) => c.is_bot);
  const humanComments = comments.filter((c) => !c.is_bot);

  return (
    <>
      <AppBar position="static" color="default" elevation={1}>
        <Toolbar>
          <IconButton edge="start" sx={{ mr: 1 }} onClick={() => navigate('/')}>
            <ArrowBackIcon />
          </IconButton>
          <Typography variant="h6" sx={{ flexGrow: 1 }} noWrap>
            {pr.repo} #{pr.pr_number}
          </Typography>
          <ReviewStatusBadge review={pr.my_review} isMyPR={pr.is_my_pr} />
        </Toolbar>
      </AppBar>

      <Container maxWidth="md" sx={{ mt: 2 }}>
        {/* PR Header */}
        <Card sx={{ mb: 2 }}>
          <CardContent>
            <Box sx={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between' }}>
              <Box>
                <Typography variant="h5" sx={{ fontWeight: 600 }}>
                  {pr.title}
                </Typography>
                <Stack direction="row" spacing={1} sx={{ mt: 1, alignItems: 'center' }}>
                  <Chip label={pr.state} size="small" color={pr.state === 'open' ? 'success' : 'default'} />
                  {pr.is_new && <Chip label="NEW" size="small" color="success" />}
                  <Typography variant="body2" color="text.secondary">
                    {pr.author} · {pr.head_branch} → {pr.base_branch}
                  </Typography>
                  <Typography variant="body2" color="text.secondary">
                    · {formatDistanceToNow(new Date(pr.created_at), { addSuffix: true })}
                  </Typography>
                </Stack>
                <Typography variant="body2" color="text.secondary" sx={{ mt: 0.5 }}>
                  {pr.changed_files_count} files changed · +{pr.additions} -{pr.deletions}
                </Typography>
              </Box>
              <Link href={pr.url} target="_blank" rel="noopener">
                <IconButton size="small"><OpenInNewIcon /></IconButton>
              </Link>
            </Box>
          </CardContent>
        </Card>

        {/* AI Summary */}
        {pr.ai_summary && (
          <Card sx={{ mb: 2 }}>
            <CardContent>
              <Typography variant="subtitle2" color="primary" gutterBottom>
                AI Summary
              </Typography>
              <ReactMarkdown>{pr.ai_summary}</ReactMarkdown>
            </CardContent>
          </Card>
        )}

        {/* Review Hints */}
        {pr.review_hints && (
          <Card sx={{ mb: 2 }}>
            <CardContent>
              <Typography variant="subtitle2" color="primary" gutterBottom>
                Key Review Areas
              </Typography>
              <ReactMarkdown>{pr.review_hints}</ReactMarkdown>
            </CardContent>
          </Card>
        )}

        {/* Risk Notes */}
        {pr.risk_notes && (
          <Card sx={{ mb: 2 }}>
            <CardContent>
              <Typography variant="subtitle2" color="warning.main" gutterBottom>
                Potential Risks
              </Typography>
              <ReactMarkdown>{pr.risk_notes}</ReactMarkdown>
            </CardContent>
          </Card>
        )}

        {/* CodeRabbit Findings */}
        {botComments.length > 0 && (
          <Card sx={{ mb: 2 }}>
            <CardContent>
              <Typography variant="subtitle2" color="info.main" gutterBottom>
                CodeRabbit Findings ({botComments.length})
              </Typography>
              {pr.coderabbit_summary && (
                <Typography variant="body2" color="text.secondary" sx={{ mb: 1 }}>
                  {pr.coderabbit_summary}
                </Typography>
              )}
              <Divider sx={{ my: 1 }} />
              {botComments.map((c) => (
                <Box key={c.id} sx={{ mb: 1.5 }}>
                  {c.file_path && (
                    <Typography variant="caption" color="text.secondary" sx={{ fontFamily: 'monospace' }}>
                      {c.file_path}{c.line ? `:${c.line}` : ''}
                    </Typography>
                  )}
                  <Typography variant="body2">{c.body.slice(0, 300)}{c.body.length > 300 ? '...' : ''}</Typography>
                </Box>
              ))}
            </CardContent>
          </Card>
        )}

        {/* Human Review Comments */}
        {humanComments.length > 0 && (
          <Card sx={{ mb: 2 }}>
            <CardContent>
              <Typography variant="subtitle2" gutterBottom>
                Review Comments ({humanComments.length})
              </Typography>
              {humanComments.map((c) => (
                <Box key={c.id} sx={{ mb: 1.5 }}>
                  <Stack direction="row" spacing={1} sx={{ alignItems: 'center' }}>
                    <Typography variant="body2" sx={{ fontWeight: 500 }}>{c.commenter}</Typography>
                    {c.file_path && (
                      <Typography variant="caption" color="text.secondary" sx={{ fontFamily: 'monospace' }}>
                        {c.file_path}{c.line ? `:${c.line}` : ''}
                      </Typography>
                    )}
                    {c.resolved && <Chip label="Resolved" size="small" color="success" sx={{ height: 18 }} />}
                  </Stack>
                  <Typography variant="body2" color="text.secondary">{c.body.slice(0, 300)}</Typography>
                </Box>
              ))}
            </CardContent>
          </Card>
        )}

        {/* My Review Status */}
        {pr.my_review && (
          <Card sx={{ mb: 2 }}>
            <CardContent>
              <Typography variant="subtitle2" gutterBottom>
                {pr.is_my_pr ? 'Your PR Status' : 'Your Review Status'}
              </Typography>
              <Stack direction="row" spacing={2} sx={{ alignItems: 'center' }}>
                <ReviewStatusBadge review={pr.my_review} isMyPR={pr.is_my_pr} />
                {!pr.is_my_pr && (
                  <Typography variant="body2" color="text.secondary">
                    State: {pr.my_review.review_state}
                  </Typography>
                )}
                {!pr.is_my_pr && pr.my_review.commits_after_review > 0 && (
                  <Typography variant="body2" color="warning.main">
                    {pr.my_review.commits_after_review} commit(s) since your review
                  </Typography>
                )}
                {pr.my_review.unresolved_comments > 0 && (
                  <Typography variant="body2" color="warning.main">
                    {pr.is_my_pr
                      ? `${pr.my_review.unresolved_comments} reviewer comment(s) to respond to`
                      : `${pr.my_review.unresolved_comments} unresolved comment(s)`}
                  </Typography>
                )}
              </Stack>
            </CardContent>
          </Card>
        )}
      </Container>
    </>
  );
}
