import { useState } from 'react'
import Box from '@mui/material/Box'
import TextField from '@mui/material/TextField'
import Button from '~/components/Button'

interface Props {
  setError: (e: string) => void
  setSuccess: (v: boolean) => void
}

export default function ContactForm({ setError, setSuccess }: Props) {
  const [email, setEmail] = useState('')
  const [desc, setDesc] = useState('')
  const [loading, setLoading] = useState(false)

  return (
    <Box
      component="form"
      sx={{
        width: '100%',
        backgroundColor: 'rgba(0, 0, 0, 0.3)',
        p: { xs: 2, lg: 4 },
        mb: 4,
        borderRadius: 2,
        'div[data-lastpass-icon-root]': { display: 'none' },
      }}
      onSubmit={(e) => {
        e.preventDefault()

        if (!email || !email.includes('@') || !email.includes('.')) {
          return setError('Email is invalid.')
        }

        if (!desc) {
          return setError(
            'Your input will allow us get prepared for the call. Please provide some information.'
          )
        }

        setLoading(true)

        fetch('/api/enquiry', {
          method: 'post',
          body: JSON.stringify({ email, desc }),
          headers: {
            'content-type': 'application/json',
          },
        })
          .then(async (res) => {
            const data = await res.json()

            if (data.ok) {
              setError('')
              return setSuccess(true)
            }

            return Promise.reject()
          })
          .catch(() => {
            setSuccess(false)
            setError(
              'Something went wrong while submitting your enquiry. Please reach out to us at hello@stormkit.io'
            )
          })
          .finally(() => {
            setLoading(false)
          })
      }}
    >
      <Box sx={{ mb: 4 }}>
        <TextField
          variant="outlined"
          label="Business email"
          name="email"
          type="email"
          placeholder="Your company email address"
          fullWidth
          autoComplete="off"
          value={email}
          onChange={(e) => {
            setEmail(e.target.value)
          }}
          InputLabelProps={{
            shrink: true,
          }}
        />
      </Box>
      <Box>
        <TextField
          multiline
          fullWidth
          maxRows={5}
          minRows={5}
          value={desc}
          onChange={(e) => {
            setDesc(e.target.value)
          }}
          name="desc"
          variant="outlined"
          label="How can we help?"
          placeholder="Help us understand better your needs and requirements"
          InputLabelProps={{
            shrink: true,
          }}
        />
      </Box>
      <Box sx={{ mt: 4 }}>
        <Button
          loading={loading}
          variant="contained"
          color="secondary"
          type="submit"
          fullWidth
        >
          Talk to Stormkit
        </Button>
      </Box>
    </Box>
  )
}
