const isDev = process.env.NODE_ENV === "development";

const apiDomain = isDev
  ? process.env.API_DOMAIN || ""
  : "https://api.stormkit.io";

export const subscriptionLink = (packageId?: string, email?: string, edition?: string): string => {
  const plan = packageId === "free" || !packageId ? "premium" : "portal";
  const params = new URLSearchParams({ plan });

  if (email) {
    params.set("ref", email);
  }

  if (edition) {
    params.set("edition", edition);
  }

  return `${apiDomain}/billing/checkout?${params}`;
};
