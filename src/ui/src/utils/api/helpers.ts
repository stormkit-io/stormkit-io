/**
 * Prepares the headers to send to github. If the access token is not set,
 * it will throw an error.
 */
export function prepareHeaders(token: string) {
  if (!token) {
    throw new Error("unauthorized");
  }

  return new Headers({
    "Content-Type": "application/json",
    Authorization: `Bearer ${token}`,
  });
}

/**
 * The error to show when the token expires.
 */
export const errTokenExpired = {
  code: 401,
  error: "Token invalid or expired.",
  reason: "token-expired",
};

/**
 * The error to show when we don't have enough permissions on the provider side.
 */
export const errNotEnoughPermissions = {
  code: 403,
  error: "We're lacking some permissions to perform this operation.",
  reason: "invalid-permissions",
};
