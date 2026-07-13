import APIKeyList from "~/shared/api-keys/APIKeyList";

interface Props {
  app: App;
  environment: Environment;
}

export default function TabAPIKey({ app, environment: env }: Props) {
  return (
    <APIKeyList
      cardId="api-keys"
      subtitle="This key will allow you to interact with our API and modify this environment."
      emptyMessage="You do not have an API key associated with this environment."
      apiKeyProps={{ appId: app.id, envId: env.id!, scope: "env" }}
    />
  );
}
