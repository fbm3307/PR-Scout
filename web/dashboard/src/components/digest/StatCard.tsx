import Card from '@mui/material/Card';
import CardContent from '@mui/material/CardContent';
import Typography from '@mui/material/Typography';
import type { SxProps } from '@mui/material/styles';

interface StatCardProps {
  title: string;
  value: number | string;
  color?: string;
  sx?: SxProps;
}

export function StatCard({ title, value, color, sx }: StatCardProps) {
  return (
    <Card sx={{ minWidth: 140, ...sx }}>
      <CardContent sx={{ textAlign: 'center', py: 2, '&:last-child': { pb: 2 } }}>
        <Typography variant="h4" sx={{ fontWeight: 700, color: color || 'primary.main' }}>
          {value}
        </Typography>
        <Typography variant="body2" color="text.secondary" sx={{ mt: 0.5 }}>
          {title}
        </Typography>
      </CardContent>
    </Card>
  );
}
