-- One-time legacy cutover: move the pre-upgrade Agro-Shield tables into an
-- archive schema. Nothing is dropped. The new migrations can then build the
-- current schema in public without colliding with the old crop/farmer model.

CREATE SCHEMA IF NOT EXISTS legacy_agro_shield;

DO $$
DECLARE
    table_name TEXT;
BEGIN
    FOREACH table_name IN ARRAY ARRAY[
        'states',
        'lga',
        'farmers',
        'crops',
        'crop_images',
        'feedback',
        'inventory_history',
        'produce_price_history',
        'deliveries',
        'notifications',
        'wallets',
        'wallet_transactions',
        'wallet_deposits',
        'wallet_transfers',
        'wallet_virtual_accounts',
        'wallet_withdrawals',
        'otp_codes',
        'diagnoses',
        'diagnosis_media',
        'conversations',
        'conversation_members',
        'chat_messages',
        'negotiations',
        'negotiation_messages',
        'orders',
        'cart_items',
        'market_events',
        'marketplace_searches',
        'external_market_prices',
        'platform_revenue',
        'bank_account_lookups',
        'webhook_events',
        'payments',
        'lga_neighbors'
    ] LOOP
        IF to_regclass(format('public.%I', table_name)) IS NOT NULL
           AND to_regclass(format('legacy_agro_shield.%I', table_name)) IS NULL THEN
            EXECUTE format(
                'ALTER TABLE public.%I SET SCHEMA legacy_agro_shield',
                table_name
            );
        END IF;
    END LOOP;
END
$$;
