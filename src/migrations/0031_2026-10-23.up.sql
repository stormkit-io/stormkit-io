-- Drop the copy of auth_wall taken before Auth Wall logins moved to teams
-- (0030). The migration shipped in v2026.10.22.1 and no rollback is needed.
DROP TABLE IF EXISTS auth_wall_backup;
