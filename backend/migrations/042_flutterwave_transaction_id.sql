-- =============================================================
-- 042_flutterwave_transaction_id.sql
--
-- Stores the provider transaction identifier returned by Flutterwave.
-- payment_reference is the reference Agro-Shield sends to Flutterwave;
-- provider_transaction_id is the identifier Flutterwave returns.
-- =============================================================

ALTER TABLE transactions
    ADD COLUMN IF NOT EXISTS provider_transaction_id TEXT;

CREATE UNIQUE INDEX IF NOT EXISTS idx_transactions_provider_transaction_id
    ON transactions(provider_transaction_id)
    WHERE provider_transaction_id IS NOT NULL;
