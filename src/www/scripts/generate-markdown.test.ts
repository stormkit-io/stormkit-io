import { describe, it } from 'node:test'
import assert from 'node:assert/strict'
import { toPublishedMarkdown } from './generate-markdown'

const source = `---
title: Getting started
description: How to deploy your first application.
---

# Welcome

Some content.
`

describe('toPublishedMarkdown', () => {
  it('strips the front matter', () => {
    const output = toPublishedMarkdown(source, '/docs/welcome/getting-started')

    assert.ok(!output.includes('---\ntitle:'))
    assert.ok(output.includes('# Welcome'))
    assert.ok(output.includes('Some content.'))
  })

  it('records the canonical url and the metadata as comments', () => {
    const output = toPublishedMarkdown(source, '/docs/welcome/getting-started')

    assert.ok(
      output.startsWith(
        '<!-- Source: https://www.stormkit.io/docs/welcome/getting-started -->'
      )
    )
    assert.ok(output.includes('<!-- Title: Getting started -->'))
    assert.ok(
      output.includes(
        '<!-- Description: How to deploy your first application. -->'
      )
    )
  })

  it('handles a file without front matter', () => {
    const output = toPublishedMarkdown('# Plain\n\nBody.\n', '/blog/plain')

    assert.ok(output.includes('# Plain'))
    assert.ok(output.endsWith('\n'))
  })
})
