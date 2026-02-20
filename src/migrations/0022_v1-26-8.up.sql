-- Migration v1_26_8: Use hashes for analytics_referrers PK to avoid index size limits

-- Add hash columns with default (hash of empty string ensures NOT NULL)
ALTER TABLE skitapi.analytics_referrers
    ADD COLUMN IF NOT EXISTS referrer_hash bytea NOT NULL DEFAULT decode(md5(''), 'hex'),
    ADD COLUMN IF NOT EXISTS request_path_hash bytea NOT NULL DEFAULT decode(md5(''), 'hex');

-- Backfill hash values for existing rows
UPDATE skitapi.analytics_referrers
    SET referrer_hash = decode(md5(referrer), 'hex'),
        request_path_hash = decode(md5(request_path), 'hex');

-- Drop old primary key and add new one using hash columns
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.table_constraints 
        WHERE table_schema = 'skitapi' AND table_name = 'analytics_referrers' AND constraint_type = 'PRIMARY KEY'
    ) THEN
        ALTER TABLE skitapi.analytics_referrers DROP CONSTRAINT IF EXISTS analytics_referrers_pkey;
    END IF;
    
    BEGIN
        ALTER TABLE skitapi.analytics_referrers ADD PRIMARY KEY (aggregate_date, referrer_hash, request_path_hash, domain_id);
    EXCEPTION WHEN duplicate_table THEN
        -- Primary key already exists
    END;
END$$;

-- Create index for efficient lookups
CREATE INDEX IF NOT EXISTS idx_analytics_referrers_hash
    ON skitapi.analytics_referrers (aggregate_date, referrer_hash, request_path_hash, domain_id);

