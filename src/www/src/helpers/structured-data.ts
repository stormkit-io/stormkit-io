import { DEFAULT_ORIGIN } from './seo'

/**
 * JSON-LD is how a crawler or an agent reads the site's identity without
 * parsing the page: who publishes it, what the product is, how to get in touch.
 * Everything here mirrors what the pages already say — nothing is asserted that
 * a visitor could not read for themselves.
 */

export const SUPPORT_EMAIL = 'hello@stormkit.io'

export const SAME_AS = [
  'https://github.com/stormkit-io',
  'https://x.com/stormkitio',
  'https://www.linkedin.com/company/stormkit',
  'https://www.youtube.com/@stormkit-io',
  'https://discord.com/invite/6yQWhyY',
]

/**
 * The registered postal address of the company. Google reads `address` as part
 * of verifying that an organisation is real.
 */
export const POSTAL_ADDRESS: Record<string, string> = {
  streetAddress: 'Sepapaja tn 6',
  postalCode: '15551',
  addressLocality: 'Tallinn',
  addressRegion: 'Harju maakond',
  addressCountry: 'EE',
}

const ORGANIZATION_ID = `${DEFAULT_ORIGIN}/#organization`

/**
 * The published price of each plan, per user per month. Both pricing tables
 * (self-hosted and Cloud) charge the same, so one list describes the product.
 */
export const PRICING_TIERS: { name: string; price: string }[] = [
  { name: 'Free', price: '0' },
  { name: 'Premium', price: '20' },
  { name: 'Ultimate', price: '100' },
]

interface JsonLd {
  '@context'?: string
  '@type': string
  [key: string]: unknown
}

export function organizationSchema(): JsonLd {
  const organization: JsonLd = {
    '@type': 'Organization',
    '@id': ORGANIZATION_ID,
    name: 'Stormkit',
    legalName: 'Stormkit',
    url: `${DEFAULT_ORIGIN}/`,
    logo: `${DEFAULT_ORIGIN}/stormkit-logo.png`,
    email: SUPPORT_EMAIL,
    description:
      'Stormkit is a self-hostable deployment platform for web applications, with deployments, environments, a managed database, end-user authentication, a mailer, cron triggers and analytics.',
    foundingDate: '2020',
    sameAs: SAME_AS,
    contactPoint: [
      {
        '@type': 'ContactPoint',
        contactType: 'customer support',
        email: SUPPORT_EMAIL,
        url: `${DEFAULT_ORIGIN}/contact`,
        availableLanguage: ['English'],
      },
      {
        '@type': 'ContactPoint',
        contactType: 'sales',
        email: SUPPORT_EMAIL,
        url: `${DEFAULT_ORIGIN}/enterprise`,
        availableLanguage: ['English'],
      },
    ],
  }

  organization.address = { '@type': 'PostalAddress', ...POSTAL_ADDRESS }

  return organization
}

export function softwareApplicationSchema(): JsonLd {
  return {
    '@type': 'SoftwareApplication',
    '@id': `${DEFAULT_ORIGIN}/#software`,
    name: 'Stormkit',
    applicationCategory: 'DeveloperApplication',
    applicationSubCategory: 'Deployment platform',
    operatingSystem: 'Linux, macOS',
    url: `${DEFAULT_ORIGIN}/`,
    downloadUrl: `${DEFAULT_ORIGIN}/install.sh`,
    softwareHelp: `${DEFAULT_ORIGIN}/docs/welcome/getting-started`,
    description:
      'Self-hostable deployment platform for web applications: git-driven deployments, preview environments, a managed Postgres database, end-user authentication, a mailer, cron triggers, volumes and analytics — driveable over a REST API and an MCP server.',
    publisher: { '@id': ORGANIZATION_ID },
    sameAs: ['https://github.com/stormkit-io/stormkit-io'],
    // Mirrors the tiers rendered by components/Pricing/PricingSelfHosted.tsx and
    // PricingCloud.tsx. Asserting a price the page contradicts is the one
    // structured-data error that costs money, so these move together.
    offers: PRICING_TIERS.map((tier) => ({
      '@type': 'Offer',
      name: tier.name,
      price: tier.price,
      priceCurrency: 'USD',
      url: `${DEFAULT_ORIGIN}/#pricing`,
    })),
  }
}

export function websiteSchema(): JsonLd {
  return {
    '@type': 'WebSite',
    '@id': `${DEFAULT_ORIGIN}/#website`,
    name: 'Stormkit',
    url: `${DEFAULT_ORIGIN}/`,
    publisher: { '@id': ORGANIZATION_ID },
  }
}

/**
 * homeStructuredData returns the graph published on the homepage: the company,
 * the product and the site itself, cross-referenced by @id so a consumer reads
 * them as one identity rather than three unrelated records.
 */
export function homeStructuredData(): string {
  return JSON.stringify({
    '@context': 'https://schema.org',
    '@graph': [
      organizationSchema(),
      softwareApplicationSchema(),
      websiteSchema(),
    ],
  })
}

/**
 * pageStructuredData returns the graph published on every other page: the
 * company plus the page itself, so identity is not homepage-only.
 */
export function pageStructuredData(params: {
  url: string
  title: string
  description: string
}): string {
  return JSON.stringify({
    '@context': 'https://schema.org',
    '@graph': [
      organizationSchema(),
      {
        '@type': 'WebPage',
        url: params.url,
        name: params.title,
        description: params.description,
        isPartOf: { '@id': `${DEFAULT_ORIGIN}/#website` },
        publisher: { '@id': ORGANIZATION_ID },
      },
    ],
  })
}

/**
 * Escapes the sequence that would end the surrounding script tag early. JSON is
 * otherwise safe to inline, but "</script>" inside a string value is not.
 */
export function toScriptTag(json: string): string {
  return `<script type="application/ld+json">${json.replace(
    /<\//g,
    '<\\/'
  )}</script>`
}
