-- Logins that may sign in with a link emailed to them. Such a login may have no
-- password, which is stored as an empty string.
ALTER TABLE auth_wall ADD COLUMN IF NOT EXISTS login_magic_link boolean DEFAULT false NOT NULL;
