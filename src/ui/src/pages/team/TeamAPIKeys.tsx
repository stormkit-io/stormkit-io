import APIKeyList from "~/shared/api-keys/APIKeyList";

interface Props {
  team: Team;
}

export default function TeamAPIKeys({ team }: Props) {
  // Team keys act on the whole team, so only owners and admins manage them.
  // Other members may only create and delete read-only keys.
  const hasWriteAccess =
    team.currentUserRole === "owner" || team.currentUserRole === "admin";

  return (
    <APIKeyList
      cardSx={{ mb: 2 }}
      subtitle={
        hasWriteAccess
          ? "This key will allow you to interact with our API and modify all apps under this team."
          : "As a developer, you can create read-only keys that fetch data for all apps under this team."
      }
      emptyMessage="You do not have an API key associated with this team yet."
      forceReadOnly={!hasWriteAccess}
      canDelete={apiKey => hasWriteAccess || !!apiKey.readOnly}
      apiKeyProps={{ teamId: team.id, scope: "team" }}
    />
  );
}
