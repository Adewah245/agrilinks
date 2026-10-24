-- Signup codes exist before an account does. Only hashes of codes and browser
-- tokens are stored. A phone has one current challenge across browsers.
CREATE TABLE registration_phone_codes (
    phone TEXT PRIMARY KEY CHECK (phone ~ '^234[0-9]{10}$'),
    token_hash TEXT NOT NULL UNIQUE CHECK (token_hash ~ '^[0-9a-f]{64}$'),
    code_hash TEXT NOT NULL CHECK (BTRIM(code_hash) <> ''),
    attempts INTEGER NOT NULL DEFAULT 0 CHECK (attempts >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMPTZ NOT NULL CHECK (expires_at > created_at),
    consumed_at TIMESTAMPTZ
);

CREATE INDEX registration_phone_codes_expiry ON registration_phone_codes(expires_at);

-- Existing accounts are not retroactively claimed to have confirmed their phone.
ALTER TABLE users ADD COLUMN phone_verified_at TIMESTAMPTZ;

CREATE FUNCTION clear_changed_phone_confirmation() RETURNS TRIGGER AS $$
BEGIN
    IF NEW.phone IS DISTINCT FROM OLD.phone THEN
        NEW.phone_verified_at := NULL;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER users_clear_changed_phone_confirmation
BEFORE UPDATE OF phone ON users
FOR EACH ROW EXECUTE FUNCTION clear_changed_phone_confirmation();
