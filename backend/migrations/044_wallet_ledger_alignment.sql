-- =============================================================
-- 044_wallet_ledger_alignment.sql
--
-- Aligns the original wallet ledger with the wallet model used by
-- the current application:
--
--   transaction_type -> type
--   reference_type/reference_id -> reference
--
-- The old reference columns are intentionally retained for
-- compatibility with historical rows. The new reference column is
-- required now because every wallet operation must be traceable by
-- a stable, unique reference.
--
-- Existing rows keep their original type value. The constraint is
-- widened rather than rewriting history, so old credit/debit rows
-- remain valid while new wallet operations can use the complete
-- ledger vocabulary.
-- =============================================================

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'wallet_transactions'
          AND column_name = 'transaction_type'
    ) AND NOT EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'wallet_transactions'
          AND column_name = 'type'
    ) THEN
        ALTER TABLE wallet_transactions
            RENAME COLUMN transaction_type TO type;
    END IF;
END
$$;

ALTER TABLE wallet_transactions
    ADD COLUMN IF NOT EXISTS reference VARCHAR(120);

UPDATE wallet_transactions
SET reference = 'legacy-wallet-txn-' || id::TEXT
WHERE reference IS NULL OR BTRIM(reference) = '';

ALTER TABLE wallet_transactions
    ALTER COLUMN reference SET NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_wallet_transactions_reference_unique
    ON wallet_transactions(reference);

ALTER TABLE wallet_transactions
    DROP CONSTRAINT IF EXISTS wallet_transactions_type_check;

ALTER TABLE wallet_transactions
    ADD CONSTRAINT wallet_transactions_type_check
    CHECK (
        type IN (
            -- Values written before this migration.
            'credit',
            'debit',

            -- Current wallet ledger operations.
            'deposit',
            'deposit_reversal',
            'withdrawal_reserved',
            'withdrawal_refund',
            'order_payment',
            'sale_credit',
            'transfer_out',
            'transfer_in',
            'transfer_refund'
        )
    );
