import { describe, it } from 'node:test'
import assert from 'node:assert/strict'
import { parseAttributes } from './markdown'

const frontMatter = (...lines: string[]) =>
  ['---', ...lines, '---', '', '# Heading', ''].join('\n')

describe('parseAttributes', () => {
  it('reads unquoted values', () => {
    const attrs = parseAttributes(frontMatter('title: Database'))
    assert.equal(attrs.title, 'Database')
  })

  it('strips the single quotes that make a colon valid YAML', () => {
    const attrs = parseAttributes(
      frontMatter("title: 'Strapi Hosting: How to Self-Host Strapi CMS'")
    )
    assert.equal(attrs.title, 'Strapi Hosting: How to Self-Host Strapi CMS')
  })

  it('strips double quotes the same way', () => {
    const attrs = parseAttributes(
      frontMatter('title: "Self-Hosting: Deploy Web Apps with Full Control"')
    )
    assert.equal(attrs.title, 'Self-Hosting: Deploy Web Apps with Full Control')
  })

  it('keeps apostrophes inside the value', () => {
    const attrs = parseAttributes(
      frontMatter("description: Improve your website's performance")
    )
    assert.equal(attrs.description, "Improve your website's performance")
  })

  it('keeps a quote that is not a matching pair', () => {
    assert.equal(parseAttributes(frontMatter("title: 'Half quoted")).title, "'Half quoted")
    assert.equal(parseAttributes(frontMatter('title: Half quoted"')).title, 'Half quoted"')
    assert.equal(parseAttributes(frontMatter('title: \'Mixed"')).title, '\'Mixed"')
  })

  it('handles a lone quote character as the value', () => {
    assert.equal(parseAttributes(frontMatter("title: '")).title, "'")
  })

  it('returns no attributes when the document has no front matter', () => {
    assert.deepEqual(parseAttributes('# Image Optimization\n\nSome text.\n'), {})
  })

  it('overrides the category when one is passed', () => {
    const attrs = parseAttributes(frontMatter('title: Database'), 'features')
    assert.equal(attrs.category, 'features')
  })
})
