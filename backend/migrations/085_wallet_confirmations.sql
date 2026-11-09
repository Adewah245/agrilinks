-- PIN is checked before this record is created. Codes belong to one exact
-- request and session; photo evidence and money commit with final consumption.
ALTER TABLE private_verification_photos DROP CONSTRAINT private_verification_photos_purpose_check;
ALTER TABLE private_verification_photos ADD CONSTRAINT private_verification_photos_purpose_check
    CHECK (purpose IN ('registration','withdrawal','transfer','legacy'));
ALTER TABLE private_photo_access_logs DROP CONSTRAINT private_photo_access_logs_purpose_check;
ALTER TABLE private_photo_access_logs ADD CONSTRAINT private_photo_access_logs_purpose_check
    CHECK (purpose IN ('registration','withdrawal','transfer','legacy'));

CREATE TABLE wallet_confirmations (
    token_hash text PRIMARY KEY CHECK (token_hash ~ '^[a-f0-9]{64}$'),
    user_id integer NOT NULL REFERENCES users(id),
    session_hash text NOT NULL,
    kind text NOT NULL CHECK (kind IN ('transfer','withdrawal')),
    currency text NOT NULL,
    amount numeric(18,2) NOT NULL CHECK (amount>0),
    recipient_wallet text NOT NULL DEFAULT '',
    note text NOT NULL DEFAULT '' CHECK (length(note)<=1000),
    quote_id text NOT NULL DEFAULT '',
    bank_code text NOT NULL DEFAULT '',
    account_number text NOT NULL DEFAULT '',
    reference text NOT NULL UNIQUE,
    otp_hash text NOT NULL,
    max_attempts integer NOT NULL CHECK (max_attempts BETWEEN 1 AND 10),
    attempts integer NOT NULL DEFAULT 0 CHECK (attempts>=0),
    expires_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    otp_sent_at timestamptz,
    verified_at timestamptz,
    invalidated_at timestamptz,
    photo_id bigint REFERENCES private_verification_photos(id),
    latitude double precision CHECK (latitude BETWEEN -90 AND 90),
    longitude double precision CHECK (longitude BETWEEN -180 AND 180),
    accuracy_m double precision CHECK (accuracy_m BETWEEN 0 AND 1000),
    captured_at timestamptz,
    received_at timestamptz,
    final_reference text,
    completed_at timestamptz,
    CHECK ((kind='transfer' AND recipient_wallet<>'' AND quote_id='' AND bank_code='' AND account_number='')
        OR (kind='withdrawal' AND recipient_wallet='' AND quote_id<>'' AND bank_code<>'' AND account_number<>'')),
    CHECK (verified_at IS NULL OR otp_sent_at IS NOT NULL),
    CHECK (final_reference IS NULL OR (verified_at IS NOT NULL AND photo_id IS NOT NULL AND completed_at IS NOT NULL)),
    CHECK (photo_id IS NULL OR (latitude IS NOT NULL AND longitude IS NOT NULL AND accuracy_m IS NOT NULL AND captured_at IS NOT NULL AND received_at IS NOT NULL))
);
CREATE INDEX wallet_confirmations_owner ON wallet_confirmations(user_id,kind,created_at DESC);
CREATE INDEX wallet_confirmations_active ON wallet_confirmations(expires_at) WHERE final_reference IS NULL AND invalidated_at IS NULL;
