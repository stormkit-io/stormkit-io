module.exports = {
  extends: ["@commitlint/config-conventional"],
  // Ignore dependabot commit messages
  ignores: [(message) => /^Bumps \[.+]\(.+\) from .+ to .+\.$/m.test(message)],
};
