import { useMemo, useState } from 'react'
import Minisearch from 'minisearch'
import Chip from '@mui/material/Chip'
import LinearProgress from '@mui/material/LinearProgress'
import SearchIcon from '@mui/icons-material/Search'
import Typography from '@mui/material/Typography'
import Modal from '@mui/material/Modal'
import Link from '@mui/material/Link'
import { grey, purple } from '@mui/material/colors'
import TextField from '@mui/material/TextField'
import Box from '@mui/material/Box'
import ArrowForwardIosIcon from '@mui/icons-material/ArrowForwardIos'
import { withContent } from '~/helpers/markdown'
import { fetchData } from '~/pages/blog/_ssr'
import data from '../../search-docs.json'

const miniSearch = new Minisearch({
  fields: ['pageTitle', 'contentTitle', 'description', 'keywords'], // fields to index for full-text search
  storeFields: ['pageTitle', 'description', 'url'], // fields to return with search results
})

miniSearch.addAll(data)

interface Props {
  onClose: () => void
}

interface Suggestion {
  pageTitle: string
  description: string
  url: string
}

interface HighlightProps {
  highlight: string
  text: string
}

function Highlight({ highlight, text }: HighlightProps) {
  // Split text on higlight term, include term itself into parts, ignore case
  const parts = useMemo(
    () => (highlight ? text.split(new RegExp(`(${highlight})`, 'gi')) : []),
    [text, highlight]
  )

  if (!highlight || highlight.length < 3) {
    return <>{text}</>
  }

  const lcHighlight = highlight.toLowerCase()

  return (
    <>
      {parts.map((part: string, index: number) => (
        <Typography component="span" key={`${part}-${index}`}>
          {part.toLowerCase().indexOf(lcHighlight) > -1 ? (
            <Typography component="span" sx={{ color: purple[500] }}>
              {part}
            </Typography>
          ) : (
            part
          )}
        </Typography>
      ))}
    </>
  )
}

interface SearchRowProps {
  title: string
  subtitle: string
  path: string
  match?: string
  onClick: () => void
}

function SearchRow({ title, subtitle, match, path, onClick }: SearchRowProps) {
  return (
    <Box
      sx={{
        p: 2,
        pb: 0,
        ':hover': { bgcolor: 'rgba(255,255,255,0.1)' },
        display: 'flex',
        alignItems: 'baseline',
        color: 'white',
      }}
    >
      <ArrowForwardIosIcon sx={{ mr: 2, fontSize: 12 }} />
      <Link
        href={path}
        sx={{
          mb: 2,
          display: 'block',
          color: 'white',
          textDecoration: 'none',
        }}
        onClick={onClick}
      >
        <Typography variant="body1">
          <Highlight highlight={match!} text={title} />
        </Typography>
        <Typography
          variant="body2"
          sx={{ color: grey[500], textTransform: 'initial' }}
        >
          <Highlight highlight={match!} text={subtitle} />
        </Typography>
      </Link>
    </Box>
  )
}

export default function DocSearchModal({ onClose }: Props) {
  const { navigation, loading } = withContent(fetchData)
  const [suggestions, setSuggestions] = useState<Suggestion[]>([])
  const [text, setText] = useState('')

  const resources = useMemo(() => {
    return navigation.filter((n) => n.search)
  }, [navigation])

  return (
    <Modal
      open
      onClose={onClose}
      sx={{
        display: 'flex',
        width: '100vw',
        height: '100vh',
        alignItems: 'center',
        justifyContent: 'center',
      }}
    >
      <Box
        sx={{
          width: { xs: '100%', md: 600 },
          height: { xs: '100%', md: 450 },
          m: 'auto',
          bgcolor: 'black',
          overflow: 'auto',
        }}
      >
        <TextField
          placeholder="Search docs"
          autoComplete="off"
          variant="outlined"
          size="medium"
          value={text}
          autoFocus
          fullWidth
          InputProps={{
            startAdornment: <SearchIcon sx={{ mx: 1, fontSize: 14 }} />,
            endAdornment: <Chip label="esc" />,
            sx: {
              fontSize: 14,
            },
          }}
          onChange={(e) => {
            const val = e.target.value
            setText(val)

            if (val.length >= 3) {
              setSuggestions(
                miniSearch
                  .search(e.target.value, { fuzzy: 3 })
                  .map((result) => ({
                    pageTitle: result.pageTitle,
                    description: result.description,
                    url: result.url,
                  }))
              )
            } else {
              setSuggestions([])
            }
          }}
        />
        {suggestions.map((sug) => (
          <SearchRow
            key={sug.url}
            path={sug.url}
            title={sug.pageTitle}
            subtitle={sug.description}
            match={text}
            onClick={() => {
              setSuggestions([])
              onClose()
            }}
          />
        ))}
        {!suggestions.length && text.length < 3 && (
          <>
            <Typography
              variant="subtitle2"
              sx={{
                textTransform: 'uppercase',
                color: 'white',
                fontSize: 12,
                fontWeight: 'bold',
                p: 2,
                borderBottom: `1px solid ${grey[900]}`,
              }}
            >
              Resources
            </Typography>
            {loading && <LinearProgress color="secondary" />}
            {resources.map((item) => (
              <SearchRow
                key={item.path}
                path={`/blog/${item.path}`}
                title={item.title}
                subtitle={item.description!}
                onClick={() => {
                  onClose()
                }}
              />
            ))}
          </>
        )}
      </Box>
    </Modal>
  )
}
