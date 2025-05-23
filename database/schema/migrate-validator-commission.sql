ALTER TABLE validator_commission
ALTER COLUMN min_self_delegation TYPE TEXT USING min_self_delegation::TEXT;