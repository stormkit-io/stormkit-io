interface APIKeyResponse {
  keys: APIKey[];
}

export default (): APIKeyResponse => ({
  keys: [
    {
      appId: "1",
      envId: "1429333243019",
      id: "8224011755",
      scope: "env",
      name: "Default",
      token:
        "SK_fkopNcPhEizN5sV7cZx69tQ1VJZSV6ZiYBdpt6qiojkOB6xbUkCfGbBjR9IxqK",
    },
    {
      appId: "1",
      envId: "1429333243019",
      id: "9868814106",
      scope: "env",
      name: "CI",
      token:
        "SK_m9nvOqzHFXiNrGEzvEQfdJK6isRCLnHpUulb5URvaSMme8fDeNUzPdhzVQLczu",
    },
  ],
});
