/**
 * Receive a repo address in Stormkit format and
 * transform it to a URL.
 */
export const toRepoAddr = (repo: string): string => {
  if (!repo) {
    return "";
  }

  const [provider, ...rest] = repo.split("/");
  const address = rest.join("/");

  if (provider === "bitbucket") {
    return `https://bitbucket.org/${address}.git`;
  }

  if (provider === "github") {
    return `https://github.com/${address}.git`;
  }

  if (provider === "gitlab") {
    return `https://gitlab.com/${address}.git`;
  }

  return "";
};

const replaceMap: Record<string, string> = {
  github: "github.com",
  gitlab: "gitlab.com",
  bitbucket: "bitbucket.org",
};

/**
 * Formats the repository address in a Stormkit format.
 */
export const formatRepo = (repo: string): string => {
  if (!repo) {
    return "";
  }

  const provider = repo
    .match(/(https:\/\/|git\@)?(.*)\.(com|org)/)?.[2]
    ?.toLowerCase();

  if (
    provider !== "github" &&
    provider !== "gitlab" &&
    provider !== "bitbucket"
  ) {
    return "";
  }

  if (repo.indexOf("git@") === 0) {
    repo = repo.replace(":", "/");
  }

  return repo
    .replace("git@", "")
    .replace(/^https:\/\//, "")
    .replace(replaceMap[provider], provider)
    .replace(/\.git$/, "");
};
