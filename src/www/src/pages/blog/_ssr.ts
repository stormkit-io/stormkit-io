import type { NavigationItem } from '~/components/DocsNav/DocsNav'
import { parseAttributes, toTitleCase } from '~/helpers/markdown'

const files = import.meta.glob('/content/blog/*.md', {
  query: '?raw',
  import: 'default',
})

interface Params {
  title?: string
}

export const fetchData: FetchDataFunc = async ({ title }: Params) => {
  let foundFile: string | undefined
  const titleLowercase = title?.toLowerCase()
  const navigation: NavigationItem[] = []
  const keys = Object.keys(files)

  for (let file of keys) {
    const fileName = file
      .replace('/content/blog/', '')
      .replace('.md', '')
      .replace(/--[\d]+/, '')

    if (fileName === titleLowercase) {
      foundFile = file
    }

    const {
      description,
      date,
      subtitle,
      title,
      authorImg,
      authorName,
      authorTw,
      search,
    } = parseAttributes((await files[file]()) as string)

    const titleNormalized = (
      title || toTitleCase(fileName.split('--')[0].replace(/-/g, ' '))
    ).replaceAll("'", '')

    navigation.push({
      path: fileName,
      title: titleNormalized,
      subtitle,
      description,
      search: search === 'true',
      date,
      active: foundFile === file,
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

  if (!foundFile || !files[foundFile]) {
    return { head: {}, context: { navigation } }
  }

  const content = (await files[foundFile]()) as string
  const attrs = parseAttributes(content)

  const index = content.indexOf('---', 2)
  const article = index > -1 ? content.slice(index + 4) : content

  return {
    head: {
      title: attrs.title?.replaceAll("'", ''),
      description: attrs.description,
      type: 'article',
    },
    context: {
      content: article,
      navigation,
    },
  }
}
