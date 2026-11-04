CREATE TABLE admin_mfa (
    user_id integer PRIMARY KEY REFERENCES users(id),
    encrypted_secret bytea NOT NULL,
    replacement_secret bytea,
    enabled boolean NOT NULL DEFAULT false,
    setup_session_hash text NOT NULL,
    setup_expires_at timestamptz NOT NULL,
    last_counter bigint NOT NULL DEFAULT -1,
    enabled_at timestamptz
);
CREATE TABLE admin_mfa_recovery_codes (
    user_id integer NOT NULL REFERENCES users(id),
    code_hash text NOT NULL,
    PRIMARY KEY(user_id,code_hash)
);
ALTER TABLE user_sessions ADD COLUMN mfa_verified_until timestamptz;
