import APIKeyList from "~/shared/api-keys/APIKeyList";

interface Props {
  user: User;
}

export default function APIKeys({ user }: Props) {
  return (
    <APIKeyList
      cardId="api-keys"
      cardSx={{ mb: 2 }}
      subtitle="This key will grant you programmatic access to everything in your Stormkit account."
      emptyMessage="You do not have an API key associated with this account."
      apiKeyProps={{ userId: user.id, scope: "user" }}
    />
  );
}
