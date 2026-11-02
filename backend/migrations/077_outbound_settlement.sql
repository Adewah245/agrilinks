ALTER TABLE provider_settlement_batches ADD COLUMN direction TEXT NOT NULL DEFAULT 'collection' CHECK(direction IN ('collection','customer_payout','provider_payable'));
ALTER TABLE provider_settlement_items DROP CONSTRAINT provider_settlement_items_check;
ALTER TABLE provider_settlement_items ADD CONSTRAINT settlement_item_net CHECK(net_kobo=gross_kobo-deductions_kobo OR net_kobo=gross_kobo+deductions_kobo);
