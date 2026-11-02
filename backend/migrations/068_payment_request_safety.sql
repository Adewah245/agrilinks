-- Claims are durable. An uncertain provider submission must be reconciled,
-- never automatically retried after a lease or application restart.
ALTER TABLE wallet_deposits ADD COLUMN checkout_claimed_at TIMESTAMPTZ;
ALTER TABLE transactions ADD COLUMN checkout_claimed_at TIMESTAMPTZ;
UPDATE wallet_deposits SET checkout_claimed_at=created_at WHERE checkout_url<>'';
UPDATE transactions SET checkout_claimed_at=created_at WHERE checkout_url<>'';

CREATE TABLE payment_provider_receipts (
    provider TEXT NOT NULL,
    provider_id TEXT NOT NULL,
    reference TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY(provider,provider_id)
);
-- Existing cross-domain conflicts are quarantined, not assigned arbitrarily.
INSERT INTO payment_provider_receipts(provider,provider_id,reference)
SELECT 'flutterwave',provider_id,
       CASE WHEN COUNT(DISTINCT reference)=1 THEN MIN(reference) ELSE 'legacy-conflict:'||provider_id END
FROM (
    SELECT provider_transaction_id AS provider_id,reference FROM wallet_deposits WHERE provider='flutterwave' AND provider_transaction_id IS NOT NULL
    UNION ALL
    SELECT provider_transaction_id,payment_reference FROM transactions WHERE payment_provider='flutterwave' AND provider_transaction_id IS NOT NULL
) receipts GROUP BY provider_id;
UPDATE wallet_deposits SET review_required=true WHERE provider_transaction_id IN
    (SELECT provider_id FROM payment_provider_receipts WHERE reference='legacy-conflict:'||provider_id);
UPDATE transactions SET review_required=true WHERE provider_transaction_id IN
    (SELECT provider_id FROM payment_provider_receipts WHERE reference='legacy-conflict:'||provider_id);
