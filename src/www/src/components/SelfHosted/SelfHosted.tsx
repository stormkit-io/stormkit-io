import { useState } from 'react'
import Box from '@mui/material/Box'
import TextField from '@mui/material/TextField'
import Typography from '@mui/material/Typography'
import Alert from '@mui/material/Alert'
import AlertTitle from '@mui/material/AlertTitle'
import Button from '@mui/material/Button'
import { grey } from '@mui/material/colors'

export default function SelfHosted() {
  const [email, setEmail] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const [success, setSuccess] = useState(false)

  return (
    <Box
      sx={{
        bgcolor: 'rgba(0,0,0,0.3)',
        p: 10,
        boxShadow: 12,
      }}
    >
      <Typography
        variant="h3"
        sx={{
          fontWeight: 600,
          fontSize: { xs: 24, md: 48 },
          textAlign: 'center',
          overflow: 'hidden',
          position: 'relative',
        }}
      >
        Need a Self-Hosted solution?
      </Typography>
      <Typography
        variant="h3"
        sx={{
          mt: 1,
          fontSize: { xs: 15, md: 18 },
          textAlign: 'center',
          opacity: 0.7,
          maxWidth: 700,
          m: 'auto',
        }}
      >
        We're here to help you setting up a Self-Hosted version on your
        infrastructure.
      </Typography>
      <Box sx={{ maxWidth: 500, m: 'auto', mt: 8 }}>
        <Box
          sx={{ display: 'flex' }}
          component="form"
          onSubmit={(e) => {
            e.preventDefault()

            if (!email || !email.includes('@') || !email.includes('.')) {
              return setError('Email is invalid.')
            }

            setLoading(true)

            fetch('/api/enquiry', {
              method: 'post',
              body: JSON.stringify({
                email,
                type: 'Self-Hosted',
              }),
              headers: {
                'content-type': 'application/json',
              },
            })
              .then(async (res) => {
                const data = await res.json()

                if (data.ok) {
                  return setSuccess(true)
                }

                return Promise.reject()
              })
              .catch(() => {
                setError(
                  'Something went wrong while submitting your enquiry. Please reach out to us at hello@stormkit.io'
                )
              })
              .finally(() => {
                setLoading(false)
              })
          }}
        >
          <TextField
            fullWidth
            variant="filled"
            type="email"
            label="Email"
            autoComplete="off"
            value={email}
            onChange={(e) => {
              setEmail(e.target.value)
            }}
          />
          <Button
            type="submit"
            variant="contained"
            color="secondary"
            loading={loading}
            sx={{ borderTopLeftRadius: 0, borderBottomLeftRadius: 0 }}
          >
            Submit
          </Button>
        </Box>

        <Typography sx={{ mt: 1, opacity: 0.7, fontSize: 14 }}>
          Leave your contact information and we will reach out to you.
        </Typography>

        {error && (
          <Alert color="error" sx={{ m: 'auto', mt: 4 }}>
            <AlertTitle>Error</AlertTitle>
            <Typography>{error}</Typography>
          </Alert>
        )}

        {success && (
          <Alert color="success" sx={{ m: 'auto', mt: 4 }}>
            <AlertTitle>Success</AlertTitle>
            <Typography>
              We received your enquiry. You will be contacted soon.
            </Typography>
          </Alert>
        )}
      </Box>
    </Box>
  )
}
