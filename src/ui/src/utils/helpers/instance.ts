/**
 * Checks whether the current instance is a self-hosted environment or not.
 *
 * @deprecated Infer the edition from context instead, which is authoritative
 * and not coupled to the URL:
 * `const isSelfHosted = details?.stormkit?.edition === "self-hosted";`
 * (`details` comes from `RootContext`). Note the edition can also be "cloud" or
 * "development", so check `=== "self-hosted"` explicitly rather than `!isCloud`.
 */
export function isSelfHosted(url: string = window.location.href): boolean {
  const regex = /^https?:\/\/(?:.*\.)?stormkit\.(io|dev)(?:\/|$)/i;
  return !regex.test(url);
}
