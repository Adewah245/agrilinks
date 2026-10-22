-- Persist the price-report domain previously held only in server memory.
-- A market/commodity series has a single unit and currency so unlike prices
-- cannot accidentally be combined in one average.
CREATE TABLE market_price_series (
    commodity_id TEXT NOT NULL CHECK (BTRIM(commodity_id) <> ''),
    market_id TEXT NOT NULL CHECK (BTRIM(market_id) <> ''),
    currency TEXT NOT NULL CHECK (currency ~ '^[A-Z]{3}$'),
    unit TEXT NOT NULL CHECK (BTRIM(unit) <> ''),
    PRIMARY KEY (commodity_id, market_id),
    UNIQUE (commodity_id, market_id, currency, unit)
);

CREATE TABLE price_submitters (
    user_id INTEGER PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    role TEXT NOT NULL CHECK (role IN ('farmer', 'field_agent', 'admin')),
    verified BOOLEAN NOT NULL DEFAULT FALSE,
    trust_score DOUBLE PRECISION NOT NULL CHECK (trust_score >= 0 AND trust_score <= 1),
    total_reports INTEGER NOT NULL DEFAULT 0 CHECK (total_reports >= 0),
    accepted_reports INTEGER NOT NULL DEFAULT 0 CHECK (accepted_reports >= 0 AND accepted_reports <= total_reports)
);

CREATE TABLE price_reports (
    id UUID PRIMARY KEY,
    commodity_id TEXT NOT NULL,
    market_id TEXT NOT NULL,
    submitter_id INTEGER NOT NULL REFERENCES users(id),
    price NUMERIC(18,2) NOT NULL CHECK (price > 0 AND price <> 'NaN'::numeric),
    currency TEXT NOT NULL,
    unit TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('pending', 'accepted', 'flagged', 'rejected')),
    flag_reason TEXT NOT NULL DEFAULT '',
    submitted_at TIMESTAMPTZ NOT NULL,
    reviewed_at TIMESTAMPTZ,
    FOREIGN KEY (commodity_id, market_id, currency, unit)
        REFERENCES market_price_series(commodity_id, market_id, currency, unit)
);
CREATE INDEX price_reports_series_time_idx ON price_reports(commodity_id, market_id, submitted_at) WHERE status='accepted';
CREATE INDEX price_reports_pending_idx ON price_reports(submitted_at) WHERE status IN ('pending', 'flagged');

CREATE TABLE price_aggregates (
    commodity_id TEXT NOT NULL,
    market_id TEXT NOT NULL,
    calculated_at TIMESTAMPTZ NOT NULL,
    avg_price DOUBLE PRECISION NOT NULL,
    min_price DOUBLE PRECISION NOT NULL,
    max_price DOUBLE PRECISION NOT NULL,
    sample_size INTEGER NOT NULL CHECK (sample_size > 0),
    PRIMARY KEY (commodity_id, market_id),
    FOREIGN KEY (commodity_id, market_id) REFERENCES market_price_series(commodity_id, market_id)
);
