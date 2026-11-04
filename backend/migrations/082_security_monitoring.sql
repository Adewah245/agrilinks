CREATE TABLE security_rate_limits (
    key_hash text PRIMARY KEY,
    started_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    attempts integer NOT NULL DEFAULT 1
);
CREATE TABLE security_events (
    id bigserial PRIMARY KEY,
    user_id integer REFERENCES users(id),
    kind text NOT NULL,
    severity text NOT NULL CHECK (severity IN ('info','warning','high')),
    created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX security_events_recent ON security_events(created_at DESC);
CREATE INDEX security_events_user ON security_events(user_id,created_at DESC);
