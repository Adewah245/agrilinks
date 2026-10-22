-- Save the currency and request key with the order. Existing orders were NGN.
-- A repeated browser submission must reuse its first order.
ALTER TABLE orders ADD COLUMN checkout_key TEXT;
ALTER TABLE orders ADD COLUMN currency VARCHAR(3) NOT NULL DEFAULT 'NGN';
CREATE UNIQUE INDEX orders_buyer_checkout_key ON orders(buyer_id, checkout_key)
    WHERE checkout_key IS NOT NULL;

-- Keep the provider link so refreshing the page does not create another charge.
ALTER TABLE transactions ADD COLUMN checkout_url TEXT NOT NULL DEFAULT '';
ALTER TABLE transactions ADD COLUMN review_required BOOLEAN NOT NULL DEFAULT FALSE;

-- Use decimal arithmetic for prices and stock, matching order_items. This
-- avoids errors such as 0.3 - 0.1 - 0.2 leaving a tiny amount of stock.
ALTER TABLE products ALTER COLUMN price TYPE NUMERIC(20,2) USING price::numeric;
ALTER TABLE products ALTER COLUMN quantity TYPE NUMERIC(20,6) USING quantity::numeric;
ALTER TABLE products ALTER COLUMN initial_listed_quantity TYPE NUMERIC(20,6) USING initial_listed_quantity::numeric;
ALTER TABLE product_price_history ALTER COLUMN price TYPE NUMERIC(20,2) USING price::numeric;
ALTER TABLE inventory_history ALTER COLUMN quantity_before TYPE NUMERIC(20,6) USING quantity_before::numeric;
ALTER TABLE inventory_history ALTER COLUMN quantity_change TYPE NUMERIC(20,6) USING quantity_change::numeric;
ALTER TABLE inventory_history ALTER COLUMN quantity_after TYPE NUMERIC(20,6) USING quantity_after::numeric;
ALTER TABLE cart_items ALTER COLUMN quantity TYPE NUMERIC(20,6) USING quantity::numeric;
ALTER TABLE negotiations ALTER COLUMN quantity TYPE NUMERIC(20,6) USING quantity::numeric;
