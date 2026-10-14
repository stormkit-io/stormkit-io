-- Read-only API keys.
--
-- A read-only key can look but not touch: the REST API answers it on GET and
-- HEAD only, and the MCP server hides and refuses tools that change state. It is
-- meant for monitors, dashboards and agents that should never deploy, delete or
-- send anything.
--
-- Existing keys keep full access, so the column defaults to false.
ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS read_only boolean DEFAULT false NOT NULL;
