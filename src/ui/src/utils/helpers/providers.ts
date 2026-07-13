import githubLogo from "~/assets/logos/github-logo.svg";
import bitbucketLogo from "~/assets/logos/bitbucket-logo.svg";
import gitlabLogo from "~/assets/logos/gitlab-logo.svg";

const logos: Record<Provider, string> = {
  github: githubLogo,
  gitlab: gitlabLogo,
  bitbucket: bitbucketLogo,
};

export function getLogoForProvider(provider: Provider): string {
  return logos[provider];
}

interface ParseRepoReturnValue {
  provider: Provider;
  repo: string;
}

// The backend sends the repo name in the following format:
//
// :provider/:owner/:repo
//
// This method parses that string and returns the `repo` and the `provider`.
export function parseRepo(repo: string): ParseRepoReturnValue {
  const pieces = repo.split("/");
  const provider = pieces.shift();

  if (
    provider !== "github" &&
    provider !== "gitlab" &&
    provider !== "bitbucket"
  ) {
    throw new Error(`Invalid provider given: ${repo}`);
  }

  return {
    provider,
    repo: pieces.join("/"),
  };
}
