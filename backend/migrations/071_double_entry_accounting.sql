CREATE TABLE company_settlement_accounts (
 id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
 company_name TEXT NOT NULL,
 bank_name TEXT NOT NULL,
 bank_code TEXT NOT NULL,
 account_number TEXT NOT NULL CHECK(account_number ~ '^[0-9]{10}$'),
 resolved_account_name TEXT NOT NULL,
 currency TEXT NOT NULL CHECK(currency='NGN'),
 created_by INTEGER NOT NULL REFERENCES users(id),
 reason TEXT NOT NULL,
 created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
 UNIQUE(bank_code,account_number)
);
CREATE TABLE accounting_accounts (
 code TEXT PRIMARY KEY,
 name TEXT NOT NULL,
 category TEXT NOT NULL CHECK(category IN ('asset','liability','revenue','expense','equity','memo_asset','memo_liability')),
 wallet_id INTEGER UNIQUE REFERENCES wallets(id),
 company_settlement_account_id BIGINT REFERENCES company_settlement_accounts(id),
 CHECK(wallet_id IS NULL OR category='liability')
);
INSERT INTO accounting_accounts(code,name,category) VALUES
 ('provider_clearing','Funds held by payment provider','asset'),
 ('opening_suspense','Opening funds awaiting external reconciliation','asset'),
 ('funding_fee_revenue','AgriLinks funding service fee revenue — company only','revenue'),
 ('withdrawal_fee_revenue','AgriLinks withdrawal service fee revenue — company only','revenue'),
 ('other_platform_revenue','Other AgriLinks company revenue','revenue'),
 ('provider_settled_cash','Reconciled cash available in the provider balance','asset'),
 ('company_transfer_reserved','Company cash reserved for approved transfer workflow','asset'),
 ('pending_provider_collections','Unconfirmed collections (memorandum)','memo_asset'),
 ('pending_customer_deposits','Customer deposits awaiting confirmation (memorandum)','memo_liability'),
 ('customer_refunds_payable','Customer refunds awaiting payment','liability'),
 ('seller_payouts_payable','Seller payouts awaiting settlement','liability'),
 ('marketplace_refunds_payable','Marketplace refunds awaiting settlement','liability'),
 ('tax_statutory_payable','Approved tax and statutory obligations','liability'),
 ('company_operating_expense','Company operating expenses','expense'),
 ('statutory_expense','Approved statutory costs','expense'),
 ('company_operating_reserve','Restricted company operating reserve','equity'),
 ('chargeback_reserve','Restricted refund and chargeback reserve','equity'),
 ('reserve_allocation','Company reserve allocations','equity'),
 ('marketplace_commission','AgriLinks marketplace commission — company only','revenue'),
 ('provider_expense','Payment provider processing costs','expense'),
 ('provider_cost_recovery','Disclosed customer contribution to provider costs','expense'),
 ('provider_payable','Provider charges awaiting settlement','liability'),
 ('funding_unmatched','Unallocated incoming customer funds','liability');
CREATE TABLE accounting_journals (
 id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
 reference TEXT NOT NULL UNIQUE,
 transaction_type TEXT NOT NULL,
 status TEXT NOT NULL DEFAULT 'posted' CHECK(status='posted'),
 fingerprint TEXT NOT NULL,
 reversal_of BIGINT UNIQUE REFERENCES accounting_journals(id),
 actor_id INTEGER REFERENCES users(id),
 evidence JSONB NOT NULL DEFAULT '{}',
 created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE accounting_postings (
 journal_id BIGINT NOT NULL REFERENCES accounting_journals(id),
 line INTEGER NOT NULL,
 account_code TEXT NOT NULL REFERENCES accounting_accounts(code),
 currency TEXT NOT NULL,
 debit_kobo BIGINT NOT NULL DEFAULT 0 CHECK(debit_kobo>=0),
 credit_kobo BIGINT NOT NULL DEFAULT 0 CHECK(credit_kobo>=0),
 PRIMARY KEY(journal_id,line),
 CHECK((debit_kobo>0 AND credit_kobo=0) OR (credit_kobo>0 AND debit_kobo=0))
);
CREATE FUNCTION enforce_balanced_journal() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE target BIGINT; bad BOOLEAN; n INTEGER;
BEGIN
 IF TG_TABLE_NAME='accounting_journals' THEN target:=NEW.id; ELSE target:=NEW.journal_id; END IF;
 SELECT count(*) INTO n FROM accounting_postings WHERE journal_id=target;
 SELECT EXISTS(SELECT 1 FROM accounting_postings WHERE journal_id=target GROUP BY currency HAVING SUM(debit_kobo::numeric)<>SUM(credit_kobo::numeric)) INTO bad;
 IF n<2 OR bad THEN RAISE EXCEPTION 'unbalanced accounting journal %',target; END IF;
 RETURN NULL;
END $$;
CREATE CONSTRAINT TRIGGER journal_must_balance AFTER INSERT ON accounting_journals DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION enforce_balanced_journal();
CREATE CONSTRAINT TRIGGER postings_must_balance AFTER INSERT ON accounting_postings DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION enforce_balanced_journal();
CREATE TRIGGER immutable_journals BEFORE UPDATE OR DELETE ON accounting_journals FOR EACH ROW EXECUTE FUNCTION prevent_financial_history_edit();
CREATE TRIGGER immutable_postings BEFORE UPDATE OR DELETE ON accounting_postings FOR EACH ROW EXECUTE FUNCTION prevent_financial_history_edit();
CREATE TRIGGER immutable_company_accounts BEFORE UPDATE OR DELETE ON company_settlement_accounts FOR EACH ROW EXECUTE FUNCTION prevent_financial_history_edit();

-- Opening customer liabilities are balanced against explicitly unverified
-- opening assets. This does not pretend to confirm a provider/bank balance.
INSERT INTO accounting_accounts(code,name,category,wallet_id)
SELECT 'wallet:'||id,'Customer wallet '||id,'liability',id FROM wallets;
WITH j AS (
 INSERT INTO accounting_journals(reference,transaction_type,fingerprint,evidence)
 SELECT 'opening-wallet:'||id,'opening_balance','migration-071',jsonb_build_object('requires_external_reconciliation',true)
 FROM wallets WHERE balance>0 RETURNING id,reference
)
INSERT INTO accounting_postings(journal_id,line,account_code,currency,debit_kobo,credit_kobo)
SELECT j.id,1,'opening_suspense',w.currency,(w.balance*100)::bigint,0 FROM j JOIN wallets w ON j.reference='opening-wallet:'||w.id
UNION ALL SELECT j.id,2,'wallet:'||w.id,w.currency,0,(w.balance*100)::bigint FROM j JOIN wallets w ON j.reference='opening-wallet:'||w.id;

INSERT INTO accounting_accounts(code,name,category)
SELECT 'withdrawal:'||reference,'Pending customer withdrawal','liability' FROM wallet_withdrawals WHERE status IN ('pending','processing');
WITH j AS (
 INSERT INTO accounting_journals(reference,transaction_type,fingerprint,evidence)
 SELECT 'opening-withdrawal:'||reference,'opening_reserve','migration-071','{"requires_external_reconciliation":true}'::jsonb FROM wallet_withdrawals WHERE status IN ('pending','processing') RETURNING id,reference
)
INSERT INTO accounting_postings(journal_id,line,account_code,currency,debit_kobo,credit_kobo)
SELECT j.id,1,'opening_suspense',w.currency,(w.amount*100)::bigint,0 FROM j JOIN wallet_withdrawals w ON j.reference='opening-withdrawal:'||w.reference
UNION ALL SELECT j.id,2,'withdrawal:'||w.reference,w.currency,0,(w.amount*100)::bigint FROM j JOIN wallet_withdrawals w ON j.reference='opening-withdrawal:'||w.reference;

INSERT INTO accounting_accounts(code,name,category)
SELECT 'transfer:'||reference,'Pending wallet transfer','liability' FROM wallet_transfers WHERE status='pending';
WITH j AS (
 INSERT INTO accounting_journals(reference,transaction_type,fingerprint,evidence)
 SELECT 'opening-transfer:'||reference,'opening_reserve','migration-071','{"requires_external_reconciliation":true}'::jsonb FROM wallet_transfers WHERE status='pending' RETURNING id,reference
)
INSERT INTO accounting_postings(journal_id,line,account_code,currency,debit_kobo,credit_kobo)
SELECT j.id,1,'opening_suspense',w.currency,(w.amount*100)::bigint,0 FROM j JOIN wallet_transfers w ON j.reference='opening-transfer:'||w.reference
UNION ALL SELECT j.id,2,'transfer:'||w.reference,w.currency,0,(w.amount*100)::bigint FROM j JOIN wallet_transfers w ON j.reference='opening-transfer:'||w.reference;
