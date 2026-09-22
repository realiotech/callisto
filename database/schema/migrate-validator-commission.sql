-- Run this by hand against any existing database (test or production) before deploying code that
-- writes to validator_commission. Existing values were always small (or the row never existed, since
-- this table was never actually backfilled before), so this is a safe, lossless type widening.
ALTER TABLE validator_commission
ALTER COLUMN min_self_delegation TYPE TEXT USING min_self_delegation::TEXT;
