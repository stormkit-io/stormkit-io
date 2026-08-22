import { describe, it } from 'node:test'
import assert from 'node:assert/strict'
import {
  homeStructuredData,
  pageStructuredData,
  organizationSchema,
  softwareApplicationSchema,
  toScriptTag,
  SUPPORT_EMAIL,
  PRICING_TIERS,
} from './structured-data'
import { DEFAULT_ORIGIN } from './seo'

interface Node {
  '@type': string
  [key: string]: unknown
}

const graphOf = (json: string): Node[] =>
  (JSON.parse(json)['@graph'] as Node[]) ?? []

const nodeOf = (json: string, type: string): Node => {
  const node = graphOf(json).find((n) => n['@type'] === type)
  assert.ok(node, `expected a ${type} node`)

  return node
}

describe('homeStructuredData', () => {
  it('is a valid schema.org graph', () => {
    const parsed = JSON.parse(homeStructuredData())

    assert.equal(parsed['@context'], 'https://schema.org')
    assert.ok(Array.isArray(parsed['@graph']))
  })

  it('describes the product as a SoftwareApplication with offers', () => {
    const software = nodeOf(homeStructuredData(), 'SoftwareApplication')

    assert.equal(software.name, 'Stormkit')
    assert.equal(software.url, `${DEFAULT_ORIGIN}/`)
    assert.ok(software.description)
    assert.ok(Array.isArray(software.offers))
    assert.ok((software.offers as unknown[]).length > 0)
  })

  it('describes the company with a contactPoint carrying an email', () => {
    const organization = nodeOf(homeStructuredData(), 'Organization')
    const contactPoints = organization.contactPoint as Node[]

    assert.equal(organization.name, 'Stormkit')
    assert.ok(Array.isArray(contactPoints))
    assert.ok(contactPoints.every((point) => point.contactType))
    assert.ok(contactPoints.some((point) => point.email === SUPPORT_EMAIL))
  })

  it('lists the company profiles as sameAs so the identity is verifiable', () => {
    const organization = nodeOf(homeStructuredData(), 'Organization')
    const sameAs = organization.sameAs as string[]

    assert.ok(sameAs.includes('https://github.com/stormkit-io'))
    assert.ok(sameAs.every((url) => url.startsWith('https://')))
  })

  it('ties the product and the site to the same organization node', () => {
    const json = homeStructuredData()
    const organization = nodeOf(json, 'Organization')
    const software = nodeOf(json, 'SoftwareApplication')
    const website = nodeOf(json, 'WebSite')

    assert.deepEqual(software.publisher, { '@id': organization['@id'] })
    assert.deepEqual(website.publisher, { '@id': organization['@id'] })
  })
})

describe('pageStructuredData', () => {
  it('describes the page and keeps the company identity', () => {
    const json = pageStructuredData({
      url: `${DEFAULT_ORIGIN}/docs/api/authentication`,
      title: 'API Authentication - Stormkit',
      description: 'How to authenticate against the Stormkit API.',
    })

    const page = nodeOf(json, 'WebPage')

    assert.equal(page.url, `${DEFAULT_ORIGIN}/docs/api/authentication`)
    assert.equal(page.name, 'API Authentication - Stormkit')
    assert.ok(nodeOf(json, 'Organization'))
  })
})

describe('organizationSchema', () => {
  it('omits address rather than inventing one', () => {
    const organization = organizationSchema()

    if ('address' in organization) {
      const address = organization.address as Node
      assert.equal(address['@type'], 'PostalAddress')
    }
  })
})

describe('softwareApplicationSchema', () => {
  it('offers every published pricing tier', () => {
    const offers = softwareApplicationSchema().offers as Node[]

    assert.deepEqual(
      offers.map((offer) => offer.price),
      PRICING_TIERS.map((tier) => tier.price)
    )
    assert.ok(offers.every((offer) => offer.priceCurrency === 'USD'))
  })

  it('points at the install script and the docs', () => {
    const software = softwareApplicationSchema()

    assert.equal(software.downloadUrl, `${DEFAULT_ORIGIN}/install.sh`)
    assert.ok(String(software.softwareHelp).startsWith(`${DEFAULT_ORIGIN}/docs`))
  })
})

describe('toScriptTag', () => {
  it('wraps the json in an ld+json script tag', () => {
    const tag = toScriptTag('{"a":1}')

    assert.ok(tag.startsWith('<script type="application/ld+json">'))
    assert.ok(tag.endsWith('</script>'))
  })

  it('escapes a closing script tag inside a value', () => {
    const tag = toScriptTag(JSON.stringify({ x: '</script><script>bad()' }))

    assert.equal(tag.match(/<\/script>/g)?.length, 1)
  })
})
