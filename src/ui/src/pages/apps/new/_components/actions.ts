import api from "~/utils/api/Api";

interface InsertRepoProps {
  provider: Provider;
  repo: string;
  teamId?: string;
}

export const insertRepo = ({
  provider,
  repo,
  teamId,
}: InsertRepoProps): Promise<App> => {
  return api
    .post<{ app: App }>("/app", { provider, repo, teamId })
    .then(({ app }) => {
      return app;
    });
};
