import { describe, it } from 'node:test'
import assert from 'node:assert/strict'
import { markdownAlternate } from './seo'

describe('markdownAlternate', () => {
  it('points the homepage at its markdown twin', () => {
    assert.equal(markdownAlternate('/'), '/index.md')
    assert.equal(markdownAlternate(''), '/index.md')
  })

  it('appends .md to markdown-backed pages', () => {
    assert.equal(
      markdownAlternate('/docs/welcome/getting-started'),
      '/docs/welcome/getting-started.md'
    )
    assert.equal(markdownAlternate('/blog/faq'), '/blog/faq.md')
    assert.equal(
      markdownAlternate('/tutorials/how-to-deploy-your-self-hosted-remix-app'),
      '/tutorials/how-to-deploy-your-self-hosted-remix-app.md'
    )
  })

  it('returns nothing for pages the build does not publish as markdown', () => {
    assert.equal(markdownAlternate('/docs'), '')
    assert.equal(markdownAlternate('/blog'), '')
    assert.equal(markdownAlternate('/tutorials'), '')
    assert.equal(markdownAlternate('/vs-vercel'), '')
    assert.equal(markdownAlternate('/mcp'), '')
  })

  it('ignores query strings, fragments and trailing slashes', () => {
    assert.equal(markdownAlternate('/blog/faq/'), '/blog/faq.md')
    assert.equal(markdownAlternate('/blog/faq?ref=hn'), '/blog/faq.md')
    assert.equal(markdownAlternate('/blog/faq#top'), '/blog/faq.md')
  })
})
