-- Preserve historical balances/fees. Only new attempts receive fee version 1.
ALTER TABLE wallet_deposits ADD COLUMN fee_version INTEGER NOT NULL DEFAULT 0;
ALTER TABLE wallet_deposits ALTER COLUMN fee_version SET DEFAULT 1;
ALTER TABLE wallet_withdrawals ADD COLUMN fee_version INTEGER NOT NULL DEFAULT 0;
ALTER TABLE wallet_withdrawals ALTER COLUMN fee_version SET DEFAULT 1;
ALTER TABLE wallets ADD COLUMN recovery_debt_kobo BIGINT NOT NULL DEFAULT 0 CHECK(recovery_debt_kobo>=0);

CREATE TABLE wallet_fee_policies (
 id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
 withdrawal_provider_bearer TEXT NOT NULL CHECK(withdrawal_provider_bearer IN ('customer','platform')),
 provider_quote_tax_bps INTEGER NOT NULL CHECK(provider_quote_tax_bps BETWEEN 0 AND 10000),
 created_by INTEGER REFERENCES users(id),
 reason TEXT NOT NULL CHECK(length(btrim(reason))>0),
 created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
INSERT INTO wallet_fee_policies(withdrawal_provider_bearer,provider_quote_tax_bps,reason)
VALUES('customer',0,'Initial AgriLinks fee policy: funding 5000 kobo; withdrawal 5000/20000/30000 kobo. Provider fees use live quotes. Configure tax only after confirming merchant fee API tax treatment.');

CREATE TABLE wallet_fee_quotes (
 id TEXT PRIMARY KEY CHECK(id ~ '^[a-f0-9]{32}$'),
 user_id INTEGER NOT NULL REFERENCES users(id),
 kind TEXT NOT NULL CHECK(kind IN ('deposit','withdrawal','funding_preview')),
 currency TEXT NOT NULL CHECK(currency='NGN'),
 amount_kobo BIGINT NOT NULL CHECK(amount_kobo>0),
 service_kobo BIGINT NOT NULL CHECK(service_kobo>=0),
 provider_kobo BIGINT NOT NULL CHECK(provider_kobo>=0),
 customer_provider_kobo BIGINT NOT NULL CHECK(customer_provider_kobo>=0),
 wallet_debit_kobo BIGINT NOT NULL CHECK(wallet_debit_kobo>=0),
 wallet_credit_kobo BIGINT NOT NULL CHECK(wallet_credit_kobo>=0),
 bank_credit_kobo BIGINT NOT NULL CHECK(bank_credit_kobo>=0),
 checkout_total_kobo BIGINT NOT NULL CHECK(checkout_total_kobo>=0),
 payment_method TEXT NOT NULL DEFAULT '',
 bank_code TEXT NOT NULL DEFAULT '',
 account_number TEXT NOT NULL DEFAULT '',
 account_name TEXT NOT NULL DEFAULT '',
 bank_name TEXT NOT NULL DEFAULT '',
 policy_id BIGINT NOT NULL REFERENCES wallet_fee_policies(id),
 provider_evidence JSONB NOT NULL,
 expires_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP+INTERVAL '10 minutes',
 created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE wallet_fee_records (
 reference TEXT PRIMARY KEY,
 wallet_id INTEGER NOT NULL REFERENCES wallets(id),
 kind TEXT NOT NULL CHECK(kind IN ('deposit','withdrawal')),
 currency TEXT NOT NULL CHECK(currency='NGN'),
 principal_kobo BIGINT NOT NULL CHECK(principal_kobo>0),
 service_kobo BIGINT NOT NULL CHECK(service_kobo>=0),
 provider_quoted_kobo BIGINT NOT NULL DEFAULT 0 CHECK(provider_quoted_kobo>=0),
 provider_actual_kobo BIGINT CHECK(provider_actual_kobo>=0),
 provider_customer_kobo BIGINT NOT NULL DEFAULT 0 CHECK(provider_customer_kobo>=0),
 wallet_credit_kobo BIGINT NOT NULL DEFAULT 0 CHECK(wallet_credit_kobo>=0),
 wallet_debit_kobo BIGINT NOT NULL DEFAULT 0 CHECK(wallet_debit_kobo>=0),
 state TEXT NOT NULL CHECK(state IN ('reserved','successful','refunded','reversed','review')),
 quote_id TEXT REFERENCES wallet_fee_quotes(id),
 policy_id BIGINT NOT NULL REFERENCES wallet_fee_policies(id),
 created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
 updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE wallet_funding_intents (
 quote_id TEXT PRIMARY KEY REFERENCES wallet_fee_quotes(id),
 user_id INTEGER NOT NULL REFERENCES users(id),
 collected_kobo BIGINT NOT NULL CHECK(collected_kobo>0),
 deposit_reference TEXT UNIQUE,
 expires_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP+INTERVAL '24 hours',
 created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE FUNCTION prevent_financial_history_edit() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN RAISE EXCEPTION 'financial history is append-only'; END $$;
CREATE TRIGGER immutable_fee_policies BEFORE UPDATE OR DELETE ON wallet_fee_policies FOR EACH ROW EXECUTE FUNCTION prevent_financial_history_edit();

ALTER TABLE wallet_transactions DROP CONSTRAINT wallet_transactions_type_check;
ALTER TABLE wallet_transactions ADD CONSTRAINT wallet_transactions_type_check CHECK(type IN
 ('credit','debit','deposit','deposit_reversal','withdrawal_reserved','withdrawal_refund','order_payment','sale_credit','transfer_out','transfer_in','transfer_refund',
 'service_fee','provider_fee','fee_refund','escrow_refund','debt_repayment'));
