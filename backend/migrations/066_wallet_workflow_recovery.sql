-- Preserve existing balances and references while recording uncertain payouts
-- and deposits that need an operator to resolve a verified reversal.
ALTER TABLE wallet_withdrawals ADD COLUMN provider_submission_state TEXT NOT NULL DEFAULT 'not_submitted';
ALTER TABLE wallet_withdrawals ADD CONSTRAINT wallet_withdrawals_submission_state_check
    CHECK (provider_submission_state IN ('not_submitted','unknown','refused','accepted'));
UPDATE wallet_withdrawals SET provider_submission_state = CASE
    WHEN provider_transfer_id IS NOT NULL OR status='paid' THEN 'accepted'
    WHEN status='processing' THEN 'unknown' ELSE 'not_submitted' END;
ALTER TABLE wallet_deposits ADD COLUMN review_required BOOLEAN NOT NULL DEFAULT false;
