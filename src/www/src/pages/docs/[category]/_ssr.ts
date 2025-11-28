import type { NavigationItem } from '~/components/DocsNav/DocsNav'
import { parseAttributes, toTitleCase } from '~/helpers/markdown'

const files = import.meta.glob('/content/docs/**/*.md', {
  query: '?raw',
  import: 'default',
})

interface Params {
  category?: string
  title?: string
}

export const fetchData: FetchDataFunc = async ({
  category = 'welcome',
  title = 'getting-started',
}: Params) => {
  let foundFile: string | undefined
  const titleLowercase = title?.toLowerCase()
  const navigation: NavigationItem[] = []

  Object.keys(files).forEach((file) => {
    const fileNameWithoutPrefix = file
      .replace('/content/docs/', '')
      .replace('.md', '')

    let [fileCategory, fileTitle] = fileNameWithoutPrefix.split('--')

    fileTitle = fileTitle?.replace(/(^\d-)+/, '')

    if (category === fileCategory && fileTitle === titleLowercase) {
      foundFile = file
    }

    navigation.push({
      path: [fileCategory, fileTitle].join('/'),
      title: toTitleCase(fileTitle.replace(/-/g, ' ')),
      category: toTitleCase(fileCategory.replace(/^_/, '')),
      active: foundFile === file,
    })
  })

  if (!foundFile || !files[foundFile]) {
    return { head: {}, context: { navigation } }
  }

  const content = (await files[foundFile]()) as string
  const attrs = parseAttributes(content, category)

  const index = content.indexOf('---', 2)
  const article = index > -1 ? content.slice(index + 4) : content

  return {
    head: {
      title: attrs.title,
      description: attrs.description,
      type: 'article',
    },
    context: {
      content: article,
      navigation,
    },
  }
}
