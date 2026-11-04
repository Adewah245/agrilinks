-- Verification photographs are never public profile images.
CREATE TABLE private_verification_photos (
    id bigserial PRIMARY KEY,
    user_id integer NOT NULL REFERENCES users(id),
    purpose text NOT NULL CHECK (purpose IN ('registration','withdrawal','legacy')),
    encrypted_image bytea NOT NULL,
    created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX private_verification_photos_owner ON private_verification_photos(user_id,purpose);

CREATE TABLE withdrawal_evidence (
    token_hash text PRIMARY KEY,
    user_id integer NOT NULL REFERENCES users(id),
    session_hash text NOT NULL,
    quote_id text NOT NULL,
    reference text NOT NULL,
    expires_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    photo_id bigint REFERENCES private_verification_photos(id),
    latitude double precision CHECK (latitude BETWEEN -90 AND 90),
    longitude double precision CHECK (longitude BETWEEN -180 AND 180),
    accuracy_m double precision CHECK (accuracy_m BETWEEN 0 AND 1000),
    captured_at timestamptz,
    received_at timestamptz,
    withdrawal_id bigint UNIQUE REFERENCES wallet_withdrawals(id),
    CHECK (photo_id IS NULL OR (latitude IS NOT NULL AND longitude IS NOT NULL AND accuracy_m IS NOT NULL AND captured_at IS NOT NULL AND received_at IS NOT NULL))
);
CREATE INDEX withdrawal_evidence_reference ON withdrawal_evidence(user_id,reference);
CREATE TABLE withdrawal_evidence_reviews (
    withdrawal_id bigint NOT NULL REFERENCES wallet_withdrawals(id),
    reviewer_id integer NOT NULL REFERENCES users(id),
    session_hash text NOT NULL,
    reviewed_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    photo_viewed_at timestamptz,
    PRIMARY KEY(withdrawal_id,reviewer_id,session_hash)
);

-- Earlier versions used the same upload directory for verification and profile
-- photos. Quarantine all of those paths; an operator cannot reliably distinguish
-- their original purpose. New public profile uploads use uploads/profiles/.
CREATE TABLE legacy_private_photo_paths (
    public_path text PRIMARY KEY,
    user_id integer NOT NULL REFERENCES users(id),
    photo_id bigint REFERENCES private_verification_photos(id),
    migrated_at timestamptz
);
INSERT INTO legacy_private_photo_paths(public_path,user_id)
SELECT photo_url,min(id) FROM users WHERE photo_url LIKE '/static/uploads/users/%' OR photo_url LIKE '/uploads/users/%' GROUP BY photo_url;
UPDATE users SET photo_url='' WHERE photo_url IN (SELECT public_path FROM legacy_private_photo_paths);
UPDATE organizations SET logo_url='' WHERE logo_url IN (SELECT public_path FROM legacy_private_photo_paths);
