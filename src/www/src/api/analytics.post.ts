import http from 'node:http'
import fetch from 'node-fetch'
import * as jwt from 'jose'
import { readBody } from './_helpers'

async function logEvent(label: string) {
  const apiSecretKey = process.env.FEEZ_API_SECRET
  const apiPublicKey = process.env.FEEZ_API_PUBLIC

  if (!apiSecretKey || !apiPublicKey) {
    return false
  }

  const jwtToken = await new jwt.SignJWT({
    apiPublicKey,
  })
    .setIssuedAt()
    .setExpirationTime('1h')
    .setProtectedHeader({ alg: 'HS256' })
    .sign(new TextEncoder().encode(apiSecretKey))

  return fetch('https://www.feez.ws/api/charts/data', {
    method: 'POST',
    body: JSON.stringify({
      chartId: '18848',
      label,
    }),
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${jwtToken}`,
    },
  })
}

function referrer(ref?: string): string {
  if (!ref) {
    return 'Direct'
  }

  if (ref.match(/google|bing|yahoo|duckduckgo/)) {
    return 'SEO'
  }

  if (ref.match(/twitter|reddit|t\.co/)) {
    return 'Social Media'
  }

  if (ref.match(/indiehackers|dev\.to|medium\.com/)) {
    return 'Blog post'
  }

  return 'Other'
}

export default async function (
  req: http.IncomingMessage,
  res: http.ServerResponse
) {
  const body = await readBody<{ referrer?: string }>(req)
  const retV = await logEvent(referrer(body.referrer))

  res.setHeader('Content-Type', 'application/json')
  res.write(JSON.stringify({ ok: retV !== false }))
  res.end()
}
