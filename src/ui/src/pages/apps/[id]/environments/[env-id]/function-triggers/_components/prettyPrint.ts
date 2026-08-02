/**
 * Formats a trigger request/response body for display. JSON payloads are
 * indented for readability, anything else (HTML, plain text) is returned as is.
 */
export function formatBody(value?: string, fallback = ""): string {
  if (!value) {
    return fallback;
  }

  try {
    return JSON.stringify(JSON.parse(value), null, 2);
  } catch {
    return value;
  }
}
