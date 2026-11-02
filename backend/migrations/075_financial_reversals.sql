ALTER TABLE wallet_withdrawals DROP CONSTRAINT wallet_withdrawals_status_check;
ALTER TABLE wallet_withdrawals ADD CONSTRAINT wallet_withdrawals_status_check CHECK(status IN ('pending','processing','paid','rejected','failed','cancelled','expired','reversed'));
ALTER TABLE order_escrows DROP CONSTRAINT order_escrows_state_check;
ALTER TABLE order_escrows ADD CONSTRAINT order_escrows_state_check CHECK(state IN ('held','disputed','released','refunded','charged_back'));
CREATE TABLE payment_reversals (
 reference TEXT PRIMARY KEY REFERENCES transactions(payment_reference),
 provider_transaction_id TEXT NOT NULL,
 amount_kobo BIGINT NOT NULL CHECK(amount_kobo>0),
 evidence JSONB NOT NULL,
 created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE TRIGGER immutable_payment_reversals BEFORE UPDATE OR DELETE ON payment_reversals FOR EACH ROW EXECUTE FUNCTION prevent_financial_history_edit();
