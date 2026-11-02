-- A posted journal cannot be enlarged in a later transaction, even by adding
-- another balanced pair. Posting and balance checking happen in one transaction.
CREATE FUNCTION restrict_posting_insert() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF NOT EXISTS(SELECT 1 FROM accounting_journals WHERE id=NEW.journal_id AND xmin::text=pg_current_xact_id()::text) THEN
  RAISE EXCEPTION 'cannot append to a posted journal';
 END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER postings_same_transaction BEFORE INSERT ON accounting_postings FOR EACH ROW EXECUTE FUNCTION restrict_posting_insert();
CREATE TRIGGER immutable_account_definitions BEFORE UPDATE OR DELETE ON accounting_accounts FOR EACH ROW EXECUTE FUNCTION prevent_financial_history_edit();
CREATE TRIGGER immutable_wallet_quotes BEFORE UPDATE OR DELETE ON wallet_fee_quotes FOR EACH ROW EXECUTE FUNCTION prevent_financial_history_edit();
CREATE TRIGGER immutable_company_quotes BEFORE UPDATE OR DELETE ON company_withdrawal_quotes FOR EACH ROW EXECUTE FUNCTION prevent_financial_history_edit();
CREATE UNIQUE INDEX settlement_statement_once ON provider_settlement_batches(provider,destination,COALESCE(company_account_id,0),statement_reference);
CREATE INDEX accounting_account_currency ON accounting_postings(account_code,currency);
CREATE INDEX accounting_journal_date ON accounting_journals(created_at);
ALTER TABLE company_withdrawals ADD CONSTRAINT complete_company_reservation CHECK(total_reserved_kobo=amount_kobo+quoted_provider_fee_kobo);
ALTER TABLE company_withdrawals ADD CONSTRAINT company_approval_required CHECK(status NOT IN ('approved','processing','successful','reversed') OR (approved_by IS NOT NULL AND approved_at IS NOT NULL));
ALTER TABLE company_settlement_accounts ADD COLUMN approved_by INTEGER REFERENCES users(id);
ALTER TABLE company_settlement_accounts ADD COLUMN verification_evidence JSONB NOT NULL DEFAULT '{}';
ALTER TABLE company_settlement_accounts ADD CONSTRAINT separate_bank_approval CHECK(approved_by IS NOT NULL AND approved_by<>created_by);
ALTER TABLE wallet_funding_intents ADD COLUMN state TEXT NOT NULL DEFAULT 'pending' CHECK(state IN ('pending','matched','expired'));
