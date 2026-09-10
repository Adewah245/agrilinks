-- Transactions were introduced before orders. The orders table now exists,
-- so enforce the relationship promised by migration 011.
ALTER TABLE transactions
    ADD CONSTRAINT fk_transactions_order
    FOREIGN KEY (order_id)
    REFERENCES orders(id)
    ON DELETE CASCADE;
