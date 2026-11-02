-- Add PIN protection without inventing PINs for existing accounts. Existing
-- members enrol from their authenticated wallet before making a debit.
CREATE TABLE user_transaction_pins (
    user_id INTEGER PRIMARY KEY REFERENCES users(id) ON DELETE RESTRICT,
    pin_hash TEXT NOT NULL CHECK (length(pin_hash) = 60),
    failed_attempts INTEGER NOT NULL DEFAULT 0 CHECK (failed_attempts BETWEEN 0 AND 5),
    blocked_until TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Successful phone confirmation is retained while the user creates their PIN.
-- Account creation still consumes the browser-bound challenge atomically.
ALTER TABLE registration_phone_codes ADD COLUMN verified_at TIMESTAMPTZ;
