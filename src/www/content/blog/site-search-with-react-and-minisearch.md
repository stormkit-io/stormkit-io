---
title: Local search with react and minisearch
description: This blog post walks you through our implementation for local search using minisearch and creating react component as search box.
date: 2023-12-16
---

We recently decided to add a search bar to our documentation page, which was long overdue. After evaluating our options, we decided to use local search for now. This post will guide you through overall implementation process.

The following GIF provides a quick demonstration of the search bar. You can try it out on our documentation site.

<div class="img-wrapper">
    <img src="/assets/blog/search_box.gif" alt="Search box component created with React" />
</div>


We initially considered using [Algolia](https://www.algolia.com/), which is widely used on many other pages and offers a generous free plan. However, we didn't want to rely on a third-party service for a simple search. So, we decided to explore local search options within the browser. There are several options available, such as [fusejs](https://www.fusejs.io/) and [lunrjs](https://lunrjs.com/), but we ultimately chose [minisearch](https://lucaong.github.io/minisearch/) because of its minimalistic design and its ability to fulfill our current needs.

Most search products operate on the same fundamental principles. We need to create doc for the search to be indexed. For minisearch, it would look something like the following code:


```javascript
const documents = [
 {
    id: 2,
    pageTitle: "Configuration",
    contentTitle: "Deployment configuration",
    description: "How to configure Stormkit to deploy your applications.",
    url: "/docs/deployments/configuration"
  },
  {
    id: 4,
    pageTitle: "Auto Deployments",
    contentTitle: "Auto Deployments",
    description: "Deploy your code automatically when its merged.",
    url: "/docs/deployments/auto-deployments"
  }
]

const miniSearch = new Minisearch({
  fields: ['pageTitle', 'contentTitle', 'description', 'keywords'], // fields to index for full-text search
  storeFields: ['pageTitle', 'description', 'url'], // fields to return with search results
})

// Index all documents
miniSearch.addAll(documents)

// Search with default options
let results = miniSearch.search('auto deployments')
```

We generate our documentation page from `.md` files, which contain meta information such as title and description. To accomplish this, we have developed a script that reads the `.md` files located under the `content/docs` directory in our codebase. The script generates a `.json` file that can be imported and sent into miniSearch.

If you would like to see the implementation details of this script, please refer to this [PR](https://github.com/stormkit-io/www-stormkit-io/pull/174) (pull request). During the building phase, the JSON file is generated and fed into the miniSearch construct. This current setup works for us. However, in the future, we may consider creating an API endpoint in Stormkit and fetching this information from there. This would allow us to dynamically change the search behavior.

For the dropdown we used React Autosuggest and applied our own styling on top of it. You can again check the [PR](https://github.com/stormkit-io/www-stormkit-io/pull/174) for more details. If that is something you need in your page feel free to extract the code that you need and use it.