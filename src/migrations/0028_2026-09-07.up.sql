-- An environment serves exactly one deployment.
--
-- Percentage-based releases let one environment point at several deployments at
-- once, with traffic split between them at request time. The feature is retired:
-- publishing has replaced whatever was published before for some time, and since
-- warm-up gating landed a publish names a single deployment.
--
-- The rows are collapsed before the column goes, because an install that used
-- splits still has several published rows per environment and nothing would say
-- which of them wins once the percentages are gone. The survivor is the one
-- carrying the most traffic, and the newest of those if they are level.
--
-- The delete is guarded so this file can be run again after it has applied, and
-- ties break on the physical row identity so that even two rows identical in
-- every column leave exactly one behind. Without that, a pair of duplicates
-- would survive and take the unique index below down with them — on someone
-- else's database, where it cannot be inspected. The identity is compared as
-- text because ordering it directly needs PostgreSQL 14.
DO $$
BEGIN
	IF EXISTS (
		SELECT 1 FROM information_schema.columns
		WHERE table_schema = current_schema()
		  AND table_name = 'deployments_published'
		  AND column_name = 'percentage_released'
	) THEN
		DELETE FROM deployments_published a
		USING deployments_published b
		WHERE a.env_id = b.env_id
		  AND (a.percentage_released, a.created_at, a.ctid::text)
			< (b.percentage_released, b.created_at, b.ctid::text);
	END IF;
END $$;

ALTER TABLE deployments_published DROP COLUMN IF EXISTS percentage_released;

-- With the column gone, one-deployment-per-environment stops being a rule the
-- code re-asserts on every publish and becomes something the database holds.
CREATE UNIQUE INDEX IF NOT EXISTS idx_deployments_published_env_id_unique
	ON deployments_published (env_id);
