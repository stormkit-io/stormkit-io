import APIKeyList from "~/shared/api-keys/APIKeyList";

interface Props {
  team: Team;
}

export default function TeamAPIKeys({ team }: Props) {
  return (
    <APIKeyList
      cardSx={{ mb: 2 }}
      subtitle="This key will allow you to interact with our API and modify all apps under this team."
      emptyMessage="You do not have an API key associated with this team yet."
      apiKeyProps={{ teamId: team.id, scope: "team" }}
    />
  );
}
