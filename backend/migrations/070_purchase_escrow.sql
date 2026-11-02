CREATE TABLE order_escrows (
 order_id INTEGER PRIMARY KEY REFERENCES orders(id),
 buyer_id INTEGER NOT NULL REFERENCES users(id),
 seller_type TEXT NOT NULL CHECK(seller_type IN ('user','organization')),
 seller_id INTEGER NOT NULL,
 payment_reference TEXT NOT NULL UNIQUE,
 currency TEXT NOT NULL,
 gross_kobo BIGINT NOT NULL CHECK(gross_kobo>0),
 commission_kobo BIGINT NOT NULL CHECK(commission_kobo>=0 AND commission_kobo<=gross_kobo),
 commission_bps INTEGER NOT NULL CHECK(commission_bps BETWEEN 0 AND 10000),
 state TEXT NOT NULL DEFAULT 'held' CHECK(state IN ('held','disputed','released','refunded')),
 seller_delivered_at TIMESTAMPTZ,
 delivery_evidence TEXT NOT NULL DEFAULT '',
 buyer_confirmed_at TIMESTAMPTZ,
 dispute_reason TEXT NOT NULL DEFAULT '',
 decided_by INTEGER REFERENCES users(id),
 decided_at TIMESTAMPTZ,
 decision_reason TEXT NOT NULL DEFAULT '',
 created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
 updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE escrow_events (
 id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
 order_id INTEGER NOT NULL REFERENCES order_escrows(order_id),
 actor_id INTEGER REFERENCES users(id),
 action TEXT NOT NULL,
 reason TEXT NOT NULL,
 created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE TRIGGER immutable_escrow_events BEFORE UPDATE OR DELETE ON escrow_events FOR EACH ROW EXECUTE FUNCTION prevent_financial_history_edit();
CREATE INDEX escrow_review_queue ON order_escrows(state,created_at);
-- Historical seller credits stay historical. They are never debited again to
-- invent escrow retroactively; only new successful payments create holds.
