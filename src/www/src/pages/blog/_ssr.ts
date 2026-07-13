import type { NavigationItem } from '~/components/DocsNav/DocsNav'
import { Attributes, parseAttributes, toTitleCase } from '~/helpers/markdown'

const files = import.meta.glob('@content/blog/*.md', {
  query: '?raw',
  import: 'default',
})

interface Params {
  title?: string
}

export const fetchData: FetchDataFunc = async ({ title }: Params) => {
  let foundFile:
    | (Attributes & { fileName: string; content: string })
    | undefined
  const slug = title?.toLowerCase()
  const navigation: NavigationItem[] = []
  const keys = Object.keys(files)

  for (let file of keys) {
    const fileName = file.replace('../../content/blog/', '').replace('.md', '')
    const content = (await files[file]()) as string
    const attrs = parseAttributes(content)
    let active = false

    const {
      description,
      date,
      subtitle,
      title,
      authorImg,
      authorName,
      authorTw,
      search,
    } = attrs

    const titleNormalized = (
      title || toTitleCase(fileName.split('--')[0].replace(/-/g, ' '))
    ).replaceAll("'", '')

    if (fileName === slug) {
      active = true
      foundFile = {
        ...attrs,
        content,
        fileName: file,
      }
    }

    navigation.push({
      path: fileName,
      title: titleNormalized,
      subtitle,
      description,
      search: search === 'true',
      date,
      active,
      author:
        authorName && authorImg && authorTw
          ? { name: authorName, img: authorImg, twitter: authorTw }
          : undefined,
    })
  }

  navigation.sort((n1, n2) => {
    const date1 = n1.date || ''
    const date2 = n2.date || ''
    return date1 < date2 ? 1 : date1 > date2 ? -1 : 0
  })

  if (!foundFile || !files[foundFile.fileName]) {
    return { head: {}, context: { navigation } }
  }

  const index = foundFile.content.indexOf('---', 2)
  const article =
    index > -1 ? foundFile.content.slice(index + 4) : foundFile.content

  return {
    head: {
      title: foundFile.title,
      description: foundFile.description,
      type: 'article',
    },
    context: {
      content: article,
      navigation,
    },
  }
}
