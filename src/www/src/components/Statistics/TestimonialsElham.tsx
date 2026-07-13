import Box from '@mui/material/Box'
import Typography from '@mui/material/Typography'
import List from '@mui/material/List'
import ListItem from '@mui/material/ListItem'
import ListItemText from '@mui/material/ListItemText'
import ListItemIcon from '@mui/material/ListItemIcon'
import CheckIcon from '@mui/icons-material/Check'
import Link from '@mui/material/Link'
import Chip from '@mui/material/Chip'
import Card from '~/components/Card'
import CardHeader from '~/components/CardHeader'

export default function TestimonialsElham() {
  return (
    <Card>
      <CardHeader>
        <Typography
          sx={{
            fontSize: 24,
            display: 'flex',
            alignItems: 'center',
          }}
        >
          <Box
            component="img"
            alt="Elham"
            src="https://cdn.prod.website-files.com/64996137a1e0a71956ea90eb/64997d15979306f7e737a767_logo.png"
            sx={{ mr: 2, height: 24 }}
          />
          <Link
            href="https://www.elham.sa/"
            target="_blank"
            rel="noreferrer noopener"
            sx={{ color: 'white' }}
          >
            Elham
          </Link>
        </Typography>
        <Box sx={{ mt: 1 }}>
          {['Enterprise', 'Education', 'SaaS', 'High Traffic'].map((i) => (
            <Chip
              key={i}
              label={i}
              size="small"
              sx={{ mr: 1, color: 'white' }}
            />
          ))}
        </Box>
      </CardHeader>
      <Box>
        <Typography sx={{ mb: 2 }}>
          As a leading learning platform in Saudi Arabia, Elham leverages
          Stormkit to efficiently host and deploy their monorepo application,
          streamlining their development workflow while maintaining high
          performance across their educational ecosystem.
        </Typography>
        <List>
          {[
            'Handling millions of monthly visitors with consistent performance',
            'Achieved 99.9% uptime across all educational portals',
          ].map((i) => (
            <ListItem key={i}>
              <ListItemIcon>
                <CheckIcon color="success" sx={{ mr: 0 }} />
              </ListItemIcon>
              <ListItemText>{i}</ListItemText>
            </ListItem>
          ))}
        </List>
      </Box>
    </Card>
  )
}
