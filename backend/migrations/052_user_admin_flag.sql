-- =============================================================
-- 052_user_admin_flag.sql
--
-- Records whether a user may approve wallet operations.
--
-- The flag is stored on the account rather than only in server
-- configuration so an administrator can be granted or revoked
-- through application data without a deployment or restart.
-- =============================================================

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS is_admin BOOLEAN NOT NULL DEFAULT FALSE;

CREATE INDEX IF NOT EXISTS idx_users_is_admin
    ON users(is_admin)
    WHERE is_admin;
