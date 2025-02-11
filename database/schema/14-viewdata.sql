CREATE OR REPLACE VIEW validator_summary AS
SELECT 
    vi.consensus_address AS address,
    COALESCE(vd.moniker, NULL) AS moniker,
    vvp.voting_power,
    COALESCE(vc.commission, NULL) AS commission,
    vs.status,
    vs.jailed
FROM 
    validator_info vi
LEFT JOIN 
    validator_voting_power vvp ON vi.consensus_address = vvp.validator_address
LEFT JOIN 
    validator_commission vc ON vi.consensus_address = vc.validator_address
LEFT JOIN 
    validator_status vs ON vi.consensus_address = vs.validator_address
LEFT JOIN 
    validator_description vd ON vi.consensus_address = vd.validator_address;

CREATE OR REPLACE VIEW account_summary AS
SELECT 
    ac.address AS address,
    COALESCE(bl.amount, NULL) AS amount,
    COALESCE(bl.denom, NULL) AS denom
FROM 
    account ac
LEFT JOIN 
    balance bl ON bl.address = ac.address;

CREATE FUNCTION get_ms_locks_sorted(
  p_staker_addr TEXT DEFAULT NULL,
  p_denom TEXT DEFAULT NULL,
  p_val_addr TEXT DEFAULT NULL,
  p_limit INT DEFAULT NULL,
  p_offset INT DEFAULT NULL,
  p_order_direction TEXT DEFAULT 'asc'
) RETURNS SETOF ms_locks AS $$
SELECT *
FROM ms_locks
WHERE 
  (p_staker_addr IS NULL OR staker_addr = p_staker_addr) AND
  (p_denom IS NULL OR denom = p_denom) AND
  (p_val_addr IS NULL OR val_addr = p_val_addr)
ORDER BY 
  CASE 
    WHEN p_order_direction = 'desc' THEN amount::NUMERIC END DESC,
  CASE 
    WHEN p_order_direction = 'asc' THEN amount::NUMERIC END ASC
LIMIT p_limit
OFFSET p_offset;
$$ LANGUAGE SQL STABLE;

CREATE FUNCTION get_ms_unlocks_sorted(
  p_staker_addr TEXT DEFAULT NULL,
  p_denom TEXT DEFAULT NULL,
  p_val_addr TEXT DEFAULT NULL,
  p_limit INT DEFAULT NULL,
  p_offset INT DEFAULT NULL,
  p_order_direction TEXT DEFAULT 'asc'
) RETURNS SETOF ms_unlocks AS $$
SELECT *
FROM ms_unlocks
WHERE 
  (p_staker_addr IS NULL OR staker_addr = p_staker_addr) AND
  (p_denom IS NULL OR denom = p_denom) AND
  (p_val_addr IS NULL OR val_addr = p_val_addr)
ORDER BY 
  CASE 
    WHEN p_order_direction = 'desc' THEN amount::NUMERIC END DESC,
  CASE 
    WHEN p_order_direction = 'asc' THEN amount::NUMERIC END ASC
LIMIT p_limit
OFFSET p_offset;
$$ LANGUAGE SQL STABLE;

CREATE OR REPLACE FUNCTION get_balance_sorted(
  p_denom TEXT DEFAULT NULL,
  p_limit INT DEFAULT NULL,
  p_offset INT DEFAULT NULL,
  p_order_direction TEXT DEFAULT 'asc'
) RETURNS SETOF balance AS $$
SELECT *
FROM balance
WHERE 
  (p_denom IS NULL OR denom = p_denom)
ORDER BY 
  CASE 
    WHEN p_order_direction = 'desc' THEN amount::NUMERIC END DESC,
  CASE 
    WHEN p_order_direction = 'asc' THEN amount::NUMERIC END ASC
LIMIT p_limit
OFFSET p_offset;
$$ LANGUAGE SQL STABLE;
