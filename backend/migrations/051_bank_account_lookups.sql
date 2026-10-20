-- =============================================================
-- 051_bank_account_lookups.sql
--
-- Cache for resolving an account number to its bank.
--
-- The provider has no reverse lookup, so resolving a number means
-- probing each bank in turn. The answer is stored once and reused.
--
-- An empty bank_code records a completed probe that found no bank;
-- that lets a mistyped number avoid repeating the full sweep on
-- every attempt.
-- =============================================================

CREATE TABLE IF NOT EXISTS bank_account_lookups (
    account_number VARCHAR(10) NOT NULL,

    bank_code TEXT NOT NULL,

    bank_name TEXT NOT NULL DEFAULT '',

    account_name TEXT NOT NULL DEFAULT '',

    probed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (account_number, bank_code),

    CONSTRAINT bank_account_lookups_account_number_check
        CHECK (account_number ~ '^[0-9]{10}$')
);

CREATE INDEX IF NOT EXISTS idx_bank_account_lookups_probed_at
    ON bank_account_lookups(probed_at);
