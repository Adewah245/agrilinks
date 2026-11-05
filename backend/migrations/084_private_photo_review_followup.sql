-- Migration 083 is already applied in existing installations and is immutable.
-- Add the photo lookup index here instead of changing that migration.
CREATE INDEX IF NOT EXISTS private_photo_access_photo
    ON private_photo_access_logs(photo_id);

-- Require fresh photo access for pending, unsubmitted reviews when upgrading.
-- Keep completed requests' existing timestamps and access logs unchanged.
UPDATE withdrawal_evidence_reviews
SET photo_viewed_at=NULL, registration_photo_viewed_at=NULL
WHERE withdrawal_id IN (
    SELECT id FROM wallet_withdrawals
    WHERE status='pending' AND provider_submission_state='not_submitted'
);
