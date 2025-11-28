import type { NavigationItem } from '~/components/DocsNav/DocsNav'
import { parseAttributes, Attributes } from '~/helpers/markdown'

const files = import.meta.glob('/content/tutorials/**/*.md', {
  query: '?raw',
  import: 'default',
})

interface Params {
  slug?: string
}

export const fetchData: FetchDataFunc = async ({ slug }: Params) => {
  let foundFile: Attributes | undefined
  let articleContent: string | undefined
  const navigation: NavigationItem[] = []

  for (const file of Object.keys(files)) {
    const content = (await files[file]()) as string
    const tutorial = parseAttributes(content)
    const fileName = file.replace('/content/tutorials/', '').replace('.md', '')

    if (slug === fileName) {
      foundFile = tutorial
      articleContent = content
    }

    navigation.push({
      path: fileName,
      title: tutorial.title!,
      subtitle: tutorial.subtitle,
      description: tutorial.description,
      date: tutorial.date,
      active: slug === fileName,
    })
  }

  if (!articleContent) {
    articleContent = 'Tutorial is not found.'
  }

  const index = articleContent.indexOf('---', 2)
  const article = index > -1 ? articleContent.slice(index + 4) : articleContent

  return {
    head: {
      title: foundFile?.title || 'Tutorial is not found',
      description: foundFile?.description || 'Tutorial is not found',
      date: foundFile?.date || Date.now().toString(),
      type: 'article',
    },
    context: {
      navigation,
      content: article,
    },
  }
}
