export const mockTriggerLog = (): TriggerLog => ({
  triggerId: "2",
  createdAt: 1734602569,
  request: {
    method: "POST",
    url: "https://app.stormkit.io/api/test",
    payload: `{"hello":"world"}`,
  },
  response: {
    code: 200,
    body: `{"status":"ok"}`,
  },
});

export default (): FunctionTrigger[] => [
  {
    id: "2",
    cron: "2 0 2 * *",
    status: true,
    options: {
      url: "https://app.stormkit.io/api/test",
      method: "POST",
      payload: "hello-world",
      headers: {},
    },
  },
];
