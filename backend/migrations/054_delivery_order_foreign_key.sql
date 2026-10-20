-- =============================================================
-- 054_delivery_order_foreign_key.sql
--
-- Migration 007 created deliveries before orders existed, so it left
-- deliveries.order_id without a foreign key. Orders now exist; connect the
-- relationship without changing any existing delivery data.
-- =============================================================

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'fk_deliveries_order'
    ) THEN
        ALTER TABLE deliveries
            ADD CONSTRAINT fk_deliveries_order
            FOREIGN KEY (order_id)
            REFERENCES orders(id)
            ON DELETE CASCADE;
    END IF;
END
$$;
