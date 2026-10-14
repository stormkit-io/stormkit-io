-- Forgejo (and Gitea) instances can be connected as a git provider.
ALTER TYPE access_token_type ADD VALUE IF NOT EXISTS 'forgejo';
