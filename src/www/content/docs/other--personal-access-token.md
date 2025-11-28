---
title: Personal access tokens
description: Manage personal access tokens for shared accounts.
---

# Personal Access Token

<div>
The personal access token is intended to work only for GitLab. If you need to use it with other providers, please <a href="https://github.com/stormkit-io/app-stormkit-io/issues/new/choose" rel="noopener noreferrer" target="_blank"> open an issue</a>.
</div>

<section>

1. Follow [this guide](https://docs.gitlab.com/ee/user/profile/personal_access_tokens.html) to create a personal access token on GitLab.
2. Make sure to have the api permission checked.
3. Once you have created the token, copy it somewhere. You won't be able to access it again.
4. Click on the menu icon on the top right of the screen and then click on Account.
5. Find GitLab under Connected Accounts, click on Set personal access token button, it will open up a modal.
6. Paste the token inside the box and hit Submit.
7. The application owner must perform this operation in order to make it work for GitLab repositories.
8. You can see the owner by clicking on your application and then navigating to the Team page.

</section>

## Deleting an existing token

<section>

1. Click on the menu icon on the top right of the screen and then click on Account.
2. Find GitLab under Connected Accounts, click on Set personal access token button, it will open up a modal.
3. Click on delete existing token button. Stormkit will switch to oAuth in the absence of a personal access token.

</section>
