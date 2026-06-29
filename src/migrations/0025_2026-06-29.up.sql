-- source distinguishes server-recorded analytics hits ("server") from
-- client-side beacon pageviews ("client"), so the two can be told apart in the
-- dashboard. Nullable with no default to avoid rewriting the table; legacy rows
-- (NULL) are treated as server.
ALTER TABLE analytics ADD COLUMN IF NOT EXISTS source text;

-- snippet_interpolate opts a snippet into request-time interpolation of defined
-- system variables (e.g. {{SK_REQUEST_ID}}) at inject time. Off by default; only
-- flagged snippets are scanned, so unflagged user content is never rewritten.
ALTER TABLE snippets ADD COLUMN IF NOT EXISTS snippet_interpolate boolean NOT NULL DEFAULT false;
