ALTER TABLE skitapi.apps_build_conf ADD COLUMN IF NOT EXISTS schema_conf BYTEA;
ALTER TABLE skitapi.deployments ADD COLUMN IF NOT EXISTS migrations_folder TEXT;
ALTER TABLE skitapi.deployments ADD COLUMN IF NOT EXISTS upload_result jsonb;
ALTER TABLE skitapi.deployments DROP COLUMN IF EXISTS s3_number_of_files;

-- Migrate data from old columns to upload_result jsonb structure
DO $$
DECLARE
    has_api_package_size boolean;
    has_client_package_size boolean;
    has_server_package_size boolean;
    has_function_location boolean;
    has_storage_location boolean;
    has_api_location boolean;
    json_parts TEXT[];
    where_parts TEXT[];
BEGIN
    -- Check which columns exist
    SELECT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_schema = 'skitapi' 
        AND table_name = 'deployments' 
        AND column_name = 'api_package_size'
    ) INTO has_api_package_size;

    SELECT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_schema = 'skitapi' 
        AND table_name = 'deployments' 
        AND column_name = 'client_package_size'
    ) INTO has_client_package_size;

    SELECT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_schema = 'skitapi' 
        AND table_name = 'deployments' 
        AND column_name = 'server_package_size'
    ) INTO has_server_package_size;

    SELECT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_schema = 'skitapi' 
        AND table_name = 'deployments' 
        AND column_name = 'function_location'
    ) INTO has_function_location;

    SELECT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_schema = 'skitapi' 
        AND table_name = 'deployments' 
        AND column_name = 'storage_location'
    ) INTO has_storage_location;

    SELECT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_schema = 'skitapi' 
        AND table_name = 'deployments' 
        AND column_name = 'api_location'
    ) INTO has_api_location;

    -- Only migrate if at least one old column exists
    IF has_api_package_size OR has_client_package_size OR has_server_package_size OR 
       has_function_location OR has_storage_location OR has_api_location THEN
        
        -- Build array of non-null JSON parts
        json_parts := ARRAY[]::TEXT[];
        where_parts := ARRAY[]::TEXT[];

        IF has_api_package_size THEN
            json_parts := array_append(json_parts, '''serverlessBytes'', api_package_size');
            where_parts := array_append(where_parts, 'api_package_size IS NOT NULL');
        END IF;

        IF has_client_package_size THEN
            json_parts := array_append(json_parts, '''clientBytes'', client_package_size');
            where_parts := array_append(where_parts, 'client_package_size IS NOT NULL');
        END IF;

        IF has_server_package_size THEN
            json_parts := array_append(json_parts, '''serverBytes'', server_package_size');
            where_parts := array_append(where_parts, 'server_package_size IS NOT NULL');
        END IF;

        IF has_function_location THEN
            json_parts := array_append(json_parts, '''serverLocation'', function_location');
            where_parts := array_append(where_parts, 'function_location IS NOT NULL');
        END IF;

        IF has_storage_location THEN
            json_parts := array_append(json_parts, '''clientLocation'', storage_location');
            where_parts := array_append(where_parts, 'storage_location IS NOT NULL');
        END IF;

        IF has_api_location THEN
            json_parts := array_append(json_parts, '''serverlessLocation'', api_location');
            where_parts := array_append(where_parts, 'api_location IS NOT NULL');
        END IF;

        -- Execute update with properly joined parts
        EXECUTE format($sql$
            UPDATE skitapi.deployments
            SET upload_result = jsonb_build_object(%s)
            WHERE upload_result IS NULL
            AND (%s)
        $sql$,
            array_to_string(json_parts, ', '),
            array_to_string(where_parts, ' OR ')
        );
        
        RAISE NOTICE 'Migrated deployment upload results from old columns to upload_result jsonb';
    ELSE
        RAISE NOTICE 'No old columns found, skipping migration';
    END IF;

    -- Drop old columns if they exist
    IF has_api_package_size THEN
        ALTER TABLE skitapi.deployments DROP COLUMN api_package_size;
        RAISE NOTICE 'Dropped column api_package_size';
    END IF;

    IF has_client_package_size THEN
        ALTER TABLE skitapi.deployments DROP COLUMN client_package_size;
        RAISE NOTICE 'Dropped column client_package_size';
    END IF;

    IF has_server_package_size THEN
        ALTER TABLE skitapi.deployments DROP COLUMN server_package_size;
        RAISE NOTICE 'Dropped column server_package_size';
    END IF;

    IF has_function_location THEN
        ALTER TABLE skitapi.deployments DROP COLUMN function_location;
        RAISE NOTICE 'Dropped column function_location';
    END IF;

    IF has_storage_location THEN
        ALTER TABLE skitapi.deployments DROP COLUMN storage_location;
        RAISE NOTICE 'Dropped column storage_location';
    END IF;

    IF has_api_location THEN
        ALTER TABLE skitapi.deployments DROP COLUMN api_location;
        RAISE NOTICE 'Dropped column api_location';
    END IF;
END $$;
