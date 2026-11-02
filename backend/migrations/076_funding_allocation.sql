ALTER TABLE wallet_fee_quotes DROP CONSTRAINT wallet_fee_quotes_kind_check;
ALTER TABLE wallet_fee_quotes ADD CONSTRAINT wallet_fee_quotes_kind_check CHECK(kind IN ('deposit','withdrawal','funding_preview','funding_allocation'));
ALTER TABLE wallet_fee_quotes ADD COLUMN allocation_reference TEXT NOT NULL DEFAULT '';
ALTER TABLE accounting_journals ADD COLUMN source TEXT NOT NULL DEFAULT 'agrilinks';
CREATE TABLE settlement_import_revisions (
 id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
 batch_id BIGINT NOT NULL REFERENCES provider_settlement_batches(id),
 imported_by INTEGER NOT NULL REFERENCES users(id),
 record JSONB NOT NULL,
 created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE TRIGGER immutable_settlement_revisions BEFORE UPDATE OR DELETE ON settlement_import_revisions FOR EACH ROW EXECUTE FUNCTION prevent_financial_history_edit();
