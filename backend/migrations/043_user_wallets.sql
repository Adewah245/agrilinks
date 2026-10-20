-- =============================================================
-- 043_user_wallets.sql
--
-- Ensures every existing user has an NGN wallet. New wallets
-- receive their permanent Agro-Shield wallet number from the
-- trigger installed by migration 040.
--
-- Wallet ownership uses the shared owner_type/owner_id model, so
-- this migration only backfills user-owned wallets. Organization
-- wallets remain a separate ownership case.
-- =============================================================

INSERT INTO wallets (owner_type, owner_id, currency)
SELECT 'user', id, 'NGN'
FROM users
ON CONFLICT (owner_type, owner_id, currency) DO NOTHING;

CREATE OR REPLACE FUNCTION create_user_wallet()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    INSERT INTO wallets (owner_type, owner_id, currency)
    VALUES ('user', NEW.id, 'NGN')
    ON CONFLICT (owner_type, owner_id, currency) DO NOTHING;

    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS trg_create_user_wallet ON users;

CREATE TRIGGER trg_create_user_wallet
    AFTER INSERT ON users
    FOR EACH ROW
    EXECUTE FUNCTION create_user_wallet();
