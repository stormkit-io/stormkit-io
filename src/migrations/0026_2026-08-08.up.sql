-- Free-form notes describing what a periodic trigger is for, rendered as
-- markdown in the UI. A trigger's cron and URL say when and where it fires, but
-- not why it exists or what breaks if it stops — that context previously lived
-- outside Stormkit.
--
-- A column rather than a key inside trigger_options: the text documents the
-- trigger, not the HTTP request that Options models, and keeping it top level
-- leaves it searchable without digging through jsonb.
--
-- Nullable with no default, so existing triggers read as "undocumented" rather
-- than carrying an empty string.
ALTER TABLE function_triggers ADD COLUMN IF NOT EXISTS documentation text;

-- Excludes a domain from visitor analytics while keeping it fully served.
-- Analytics are reported per domain, so a secondary hostname (a www alias, a
-- staging host) shows up as its own entry in the domain picker and in Team
-- Insights and keeps accumulating rows nobody reads.
--
-- Access logs deliberately stay unfiltered: they are the raw request record used
-- for debugging and billing, so an operator must still see hits on an excluded
-- domain.
--
-- NOT NULL with a false default, so every existing domain keeps being tracked.
ALTER TABLE domains ADD COLUMN IF NOT EXISTS analytics_excluded boolean DEFAULT false NOT NULL;
