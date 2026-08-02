/**
 * Maps a trigger response status code to a Span color. A missing code means the
 * request never completed, which reads as a failure rather than a neutral code.
 */
export function statusColor(code?: number): "success" | "default" | "failure" {
  if (code?.toString()?.[0] === "2") {
    return "success";
  }

  return code ? "default" : "failure";
}
