/**
 * Returns an env link given the env and app objects.
 */
export const getEnvironmentUrl = (env: Environment, app: App): string => {
  return `/app/${app.id}/envs/${env.env.toLowerCase().replace(" ", "-")}`;
};
