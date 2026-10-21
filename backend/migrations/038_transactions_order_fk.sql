-- Transactions were introduced before orders. The orders table now exists,
-- so enforce the relationship promised by migration 011. Migration 021 may
-- already have created this constraint, so leave it untouched when present.
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'fk_transactions_order'
    ) THEN
        ALTER TABLE transactions
            ADD CONSTRAINT fk_transactions_order
            FOREIGN KEY (order_id)
            REFERENCES orders(id)
            ON DELETE CASCADE;
    END IF;
END
$$;
