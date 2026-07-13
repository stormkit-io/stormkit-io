import LocalStorage from "~/utils/storage/LocalStorage";

export type RecentApp = Pick<App, "id" | "displayName" | "repo" | "isBare">;

const STORAGE_KEY = "recent_apps";
const MAX_RECENT_APPS = 10;

// Entries with neither a repo nor the isBare flag were recorded by an
// older version of this feature and would render as bare apps; drop them.
export const recentApps = (): RecentApp[] =>
  (LocalStorage.get<RecentApp[]>(STORAGE_KEY, []) || []).filter(
    a => a.repo || a.isBare,
  );

export const recordRecentApp = (app: App) => {
  const entry: RecentApp = {
    id: app.id,
    displayName: app.displayName,
    repo: app.repo,
    isBare: app.isBare,
  };

  const visited = [entry, ...recentApps().filter(a => a.id !== app.id)];

  LocalStorage.set(STORAGE_KEY, visited.slice(0, MAX_RECENT_APPS));
};
