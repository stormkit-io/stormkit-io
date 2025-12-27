interface AppProps {
  id?: string;
  repo?: string;
  displayName?: string;
}

const defaultProps = {
  id: "1",
  repo: "gitlab/stormkit-io/frontend",
  displayName: "app",
};

export default ({ repo, displayName, id }: AppProps = {}): App => ({
  id: id || defaultProps.id,
  repo: repo || defaultProps.repo,
  teamId: "231",
  displayName: displayName || defaultProps.displayName,
  createdAt: 1551184215,
  defaultEnv: "production",
  defaultEnvId: "1",
  userId: "1",
});
