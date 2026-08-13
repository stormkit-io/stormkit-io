import { describe, it } from 'node:test'
import assert from 'node:assert/strict'
import { canonicalUrl, DEFAULT_ORIGIN } from './seo'

describe('canonicalUrl', () => {
  it('returns the origin with a trailing slash for the root', () => {
    assert.equal(canonicalUrl('/'), `${DEFAULT_ORIGIN}/`)
  })

  it('keeps a normal path as-is', () => {
    assert.equal(canonicalUrl('/vs-netlify'), `${DEFAULT_ORIGIN}/vs-netlify`)
  })

  it('drops the trailing slash so /page and /page/ do not split signals', () => {
    assert.equal(canonicalUrl('/blog/'), `${DEFAULT_ORIGIN}/blog`)
    assert.equal(canonicalUrl('/blog///'), `${DEFAULT_ORIGIN}/blog`)
  })

  it('strips query strings and fragments', () => {
    assert.equal(canonicalUrl('/blog?ref=hn'), `${DEFAULT_ORIGIN}/blog`)
    assert.equal(canonicalUrl('/blog#top'), `${DEFAULT_ORIGIN}/blog`)
    assert.equal(canonicalUrl('/blog?a=1#top'), `${DEFAULT_ORIGIN}/blog`)
  })

  it('falls back to the root for an empty pathname', () => {
    assert.equal(canonicalUrl(''), `${DEFAULT_ORIGIN}/`)
  })

  it('honours an explicit origin and never doubles the slash', () => {
    assert.equal(canonicalUrl('/docs', 'https://example.com'), 'https://example.com/docs')
    assert.equal(canonicalUrl('/docs', 'https://example.com/'), 'https://example.com/docs')
  })
})
