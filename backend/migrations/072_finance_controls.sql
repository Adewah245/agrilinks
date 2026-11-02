ALTER TABLE users DROP CONSTRAINT users_staff_role_check;
ALTER TABLE users ADD CONSTRAINT users_staff_role_check CHECK(staff_role IN ('','super_admin','customer_care','operations_manager','finance_admin','senior_finance_admin'));

ALTER TABLE wallet_fee_policies ADD COLUMN funding_service_kobo BIGINT NOT NULL DEFAULT 5000 CHECK(funding_service_kobo>=0);
ALTER TABLE wallet_fee_policies ADD COLUMN withdrawal_low_kobo BIGINT NOT NULL DEFAULT 5000 CHECK(withdrawal_low_kobo>=0);
ALTER TABLE wallet_fee_policies ADD COLUMN withdrawal_mid_kobo BIGINT NOT NULL DEFAULT 20000 CHECK(withdrawal_mid_kobo>=0);
ALTER TABLE wallet_fee_policies ADD COLUMN withdrawal_high_kobo BIGINT NOT NULL DEFAULT 30000 CHECK(withdrawal_high_kobo>=0);
ALTER TABLE wallet_fee_policies ADD COLUMN effective_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP;
ALTER TABLE wallet_fee_policies ADD COLUMN approved_by INTEGER REFERENCES users(id);
ALTER TABLE wallet_fee_policies ADD CONSTRAINT fee_policy_separation CHECK(created_by IS NULL OR (approved_by IS NOT NULL AND approved_by<>created_by));

CREATE TABLE finance_change_requests (
 id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
 request_key TEXT NOT NULL UNIQUE CHECK(request_key ~ '^[a-f0-9]{32}$'),
 kind TEXT NOT NULL CHECK(kind IN ('fee_policy','company_bank','transfer_limits','reserve','statutory_obligation','opening_reconciliation')),
 payload JSONB NOT NULL,
 reason TEXT NOT NULL,
 initiated_by INTEGER NOT NULL REFERENCES users(id),
 approved_by INTEGER REFERENCES users(id),
 decision_reason TEXT NOT NULL DEFAULT '',
 status TEXT NOT NULL DEFAULT 'pending' CHECK(status IN ('pending','approved','rejected')),
 created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
 decided_at TIMESTAMPTZ,
 CHECK(approved_by IS NULL OR approved_by<>initiated_by)
);
CREATE TABLE finance_transfer_policies (
 id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
 per_transfer_limit_kobo BIGINT NOT NULL CHECK(per_transfer_limit_kobo>0),
 daily_limit_kobo BIGINT NOT NULL CHECK(daily_limit_kobo>=per_transfer_limit_kobo),
 senior_threshold_kobo BIGINT NOT NULL CHECK(senior_threshold_kobo>0),
 company_account_id BIGINT REFERENCES company_settlement_accounts(id),
 change_id BIGINT UNIQUE REFERENCES finance_change_requests(id),
 created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
INSERT INTO finance_transfer_policies(per_transfer_limit_kobo,daily_limit_kobo,senior_threshold_kobo) VALUES(100000000,1000000000,10000000);
CREATE TRIGGER immutable_transfer_policies BEFORE UPDATE OR DELETE ON finance_transfer_policies FOR EACH ROW EXECUTE FUNCTION prevent_financial_history_edit();

CREATE TABLE company_withdrawal_quotes (
 id TEXT PRIMARY KEY CHECK(id ~ '^[a-f0-9]{32}$'),
 initiated_by INTEGER NOT NULL REFERENCES users(id),
 amount_kobo BIGINT NOT NULL CHECK(amount_kobo>0),
 provider_fee_kobo BIGINT NOT NULL CHECK(provider_fee_kobo>=0),
 total_kobo BIGINT NOT NULL CHECK(total_kobo=amount_kobo+provider_fee_kobo),
 company_account_id BIGINT NOT NULL REFERENCES company_settlement_accounts(id),
 policy_id BIGINT NOT NULL REFERENCES finance_transfer_policies(id),
 provider_evidence JSONB NOT NULL,
 expires_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP+INTERVAL '10 minutes'
);
CREATE TABLE company_withdrawals (
 id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
 reference TEXT NOT NULL UNIQUE,
 quote_id TEXT NOT NULL UNIQUE REFERENCES company_withdrawal_quotes(id),
 currency TEXT NOT NULL DEFAULT 'NGN' CHECK(currency='NGN'),
 amount_kobo BIGINT NOT NULL CHECK(amount_kobo>0),
 quoted_provider_fee_kobo BIGINT NOT NULL CHECK(quoted_provider_fee_kobo>=0),
 actual_provider_fee_kobo BIGINT CHECK(actual_provider_fee_kobo>=0),
 total_reserved_kobo BIGINT NOT NULL,
 company_account_id BIGINT NOT NULL REFERENCES company_settlement_accounts(id),
 initiated_by INTEGER NOT NULL REFERENCES users(id),
 approved_by INTEGER REFERENCES users(id),
 reason TEXT NOT NULL,
 approval_reason TEXT NOT NULL DEFAULT '',
 status TEXT NOT NULL DEFAULT 'pending' CHECK(status IN ('pending','approved','processing','successful','failed','reversed')),
 provider_transfer_id TEXT UNIQUE,
 provider_submission_state TEXT NOT NULL DEFAULT 'not_submitted' CHECK(provider_submission_state IN ('not_submitted','unknown','accepted','refused')),
 created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
 approved_at TIMESTAMPTZ,
 processed_at TIMESTAMPTZ,
 verified_at TIMESTAMPTZ,
 CHECK(approved_by IS NULL OR approved_by<>initiated_by)
);
CREATE TABLE finance_action_events (
 id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
 reference TEXT NOT NULL,
 actor_id INTEGER REFERENCES users(id),
 action TEXT NOT NULL,
 reason TEXT NOT NULL,
 evidence JSONB NOT NULL DEFAULT '{}',
 created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE TRIGGER immutable_finance_events BEFORE UPDATE OR DELETE ON finance_action_events FOR EACH ROW EXECUTE FUNCTION prevent_financial_history_edit();

CREATE TABLE provider_transfer_receipts (
 provider TEXT NOT NULL,
 transfer_id TEXT NOT NULL,
 reference TEXT NOT NULL,
 PRIMARY KEY(provider,transfer_id)
);
INSERT INTO provider_transfer_receipts SELECT 'flutterwave',provider_transfer_id,reference FROM wallet_withdrawals WHERE provider_transfer_id IS NOT NULL;

CREATE TABLE provider_settlement_batches (
 id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
 provider TEXT NOT NULL,
 provider_reference TEXT NOT NULL,
 currency TEXT NOT NULL,
 gross_kobo BIGINT NOT NULL CHECK(gross_kobo>=0),
 deductions_kobo BIGINT NOT NULL CHECK(deductions_kobo>=0),
 net_kobo BIGINT NOT NULL CHECK(net_kobo>=0),
 destination TEXT NOT NULL CHECK(destination IN ('company_bank','provider_balance')),
 status TEXT NOT NULL CHECK(status IN ('imported','matched','mismatch','reversed')),
 source TEXT NOT NULL,
 evidence JSONB NOT NULL,
 imported_by INTEGER NOT NULL REFERENCES users(id),
 approved_by INTEGER REFERENCES users(id),
 statement_reference TEXT NOT NULL,
 statement_amount_kobo BIGINT NOT NULL,
 company_account_id BIGINT REFERENCES company_settlement_accounts(id),
 reason TEXT NOT NULL,
 created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
 UNIQUE(provider,provider_reference),
 CHECK(approved_by IS NULL OR approved_by<>imported_by)
);
CREATE TABLE provider_settlement_items (
 batch_id BIGINT NOT NULL REFERENCES provider_settlement_batches(id),
 reference TEXT NOT NULL UNIQUE,
 gross_kobo BIGINT NOT NULL CHECK(gross_kobo>=0),
 deductions_kobo BIGINT NOT NULL CHECK(deductions_kobo>=0),
 net_kobo BIGINT NOT NULL,
 PRIMARY KEY(batch_id,reference),
 CHECK(net_kobo=gross_kobo-deductions_kobo)
);
