CREATE TABLE company_transfer_confirmations (
 withdrawal_id BIGINT PRIMARY KEY REFERENCES company_withdrawals(id),
 company_account_id BIGINT NOT NULL REFERENCES company_settlement_accounts(id),
 statement_reference TEXT NOT NULL,
 received_kobo BIGINT NOT NULL CHECK(received_kobo>0),
 confirmed_by INTEGER NOT NULL REFERENCES users(id),
 evidence JSONB NOT NULL,
 created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
 UNIQUE(company_account_id,statement_reference)
);
CREATE TRIGGER immutable_company_confirmations BEFORE UPDATE OR DELETE ON company_transfer_confirmations FOR EACH ROW EXECUTE FUNCTION prevent_financial_history_edit();
