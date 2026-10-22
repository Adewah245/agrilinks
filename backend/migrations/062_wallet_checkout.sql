-- A saved link lets a repeated funding request use the same provider checkout.
ALTER TABLE wallet_deposits ADD COLUMN checkout_url TEXT NOT NULL DEFAULT '';
