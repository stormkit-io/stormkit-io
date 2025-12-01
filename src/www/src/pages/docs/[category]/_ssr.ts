import type { NavigationItem } from '~/components/DocsNav/DocsNav'
import { parseAttributes, toTitleCase } from '~/helpers/markdown'

const files = import.meta.glob('@docs/**/*.md', {
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

  // Example structure:
  // ../../docs/api/1-authentication.md
  Object.keys(files).forEach((fileNameRelPath) => {
    const relPath = fileNameRelPath.split('/docs/')[1] // api/1-authentication.md
    const fileCtgr = relPath.split('/')[0] // api
    const fileName = relPath
      .split('/')[1]
      .replace(/.md$/, '') // Remove the extension
      .replace(/^\d+-/, '') // Replace the sort prefix

    let isActive = false

    if (category === fileCtgr && fileName === titleLowercase) {
      foundFile = fileNameRelPath
      isActive = true
    }

    navigation.push({
      path: [fileCtgr, fileName].join('/'),
      title: toTitleCase(fileName.replace(/-/g, ' ')),
      category: toTitleCase(fileCtgr),
      active: isActive,
    })
  })

  if (!foundFile) {
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
