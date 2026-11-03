-- Rename the platform's price-history source without changing applied migrations.
ALTER TABLE product_price_history
    ALTER COLUMN source SET DEFAULT 'Agrilinks Nigeria';

UPDATE product_price_history
SET source = 'Agrilinks Nigeria'
WHERE source = 'Agro-Shield';
