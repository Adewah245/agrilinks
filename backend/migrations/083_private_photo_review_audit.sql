-- Bind each review to the account photograph shown alongside the withdrawal
-- selfie. Existing reviews must be reopened before they can authorize payout.
ALTER TABLE withdrawal_evidence_reviews
    ADD COLUMN registration_photo_id bigint REFERENCES private_verification_photos(id),
    ADD COLUMN registration_photo_viewed_at timestamptz;
UPDATE withdrawal_evidence_reviews SET photo_viewed_at=NULL;

-- Keep the access record when retention removes the image itself. No image
-- bytes, coordinates, raw session tokens, or authenticator secrets belong here.
CREATE TABLE private_photo_access_logs (
    id bigserial PRIMARY KEY,
    photo_id bigint REFERENCES private_verification_photos(id) ON DELETE SET NULL,
    owner_id integer NOT NULL REFERENCES users(id),
    viewer_id integer NOT NULL REFERENCES users(id),
    session_hash text NOT NULL,
    purpose text NOT NULL CHECK (purpose IN ('registration','withdrawal','legacy')),
    viewed_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX private_photo_access_owner ON private_photo_access_logs(owner_id,viewed_at DESC);
CREATE INDEX private_photo_access_viewer ON private_photo_access_logs(viewer_id,viewed_at DESC);
