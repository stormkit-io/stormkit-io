import http from 'node:http'
import fetch from 'node-fetch'
import { readBody } from './_helpers'

export default async function (req: http.IncomingMessage) {
  const body = await readBody<{ email?: string; desc?: string }>(req)

  if (!body.email) {
    return {
      status: 400,
      body: {
        error: "'email' is a required field.",
      },
    }
  }

  if (!body.desc) {
    return {
      status: 400,
      body: {
        error: "'desc' is a required field.",
      },
    }
  }

  const enquiryChannelUrl = process.env.DISCORD_ENQUIRY_CHANNEL_URL

  if (!enquiryChannelUrl) {
    return { body: { error: 'Channel URL is not provided', status: 500 } }
  }

  try {
    await fetch(enquiryChannelUrl, {
      method: 'POST',
      body: JSON.stringify({
        embeds: [
          {
            title: 'New Enquiry',
            timestamp: new Date().toISOString(),
            fields: [
              { name: 'Email', value: body.email },
              { name: 'Description', value: body.desc },
            ],
          },
        ],
      }),
      headers: {
        'Content-Type': 'application/json',
      },
    })

    return {
      status: 200,
      body: { ok: true },
    }
  } catch {
    return {
      status: 500,
      body: {
        error:
          'Something went wrong while submitting the enquiry. Please reach out us at hello@stormkit.io.',
      },
    }
  }
}
