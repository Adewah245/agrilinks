-- Archived tables may have constraints with the same names. Check the current
-- table itself so an old constraint cannot hide a missing current relationship.
DO $$
DECLARE link RECORD;
BEGIN
    FOR link IN SELECT * FROM (VALUES
        ('deliveries','fk_deliveries_order'),
        ('transactions','fk_transactions_order'),
        ('product_reviews','fk_product_reviews_order'),
        ('user_reviews','fk_user_reviews_order')
    ) AS relationships(table_name,constraint_name)
    LOOP
        IF NOT EXISTS (SELECT 1 FROM pg_constraint
            WHERE conrelid=format('public.%I',link.table_name)::regclass
              AND conname=link.constraint_name) THEN
            EXECUTE format('ALTER TABLE public.%I ADD CONSTRAINT %I FOREIGN KEY(order_id) REFERENCES public.orders(id) ON DELETE RESTRICT',link.table_name,link.constraint_name);
        END IF;
    END LOOP;
END $$;
