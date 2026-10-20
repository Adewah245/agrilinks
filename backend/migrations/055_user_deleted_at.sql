-- =============================================================
-- 055_user_deleted_at.sql
--
-- Marks a user account as closed without deleting the users row.
-- Orders, transactions, deliveries, and wallet history keep a
-- stable user id, while active-account queries exclude the row.
--
-- The phone number is replaced with a unique placeholder during
-- closure, so the original number can be used by a future signup.
-- The email column is already nullable and is cleared at the same
-- time.
-- =============================================================

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP;

CREATE INDEX IF NOT EXISTS idx_users_deleted_at
    ON users(deleted_at)
    WHERE deleted_at IS NOT NULL;
