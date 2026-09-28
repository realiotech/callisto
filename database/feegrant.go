package database

import (
	"fmt"
	"time"

	"github.com/cosmos/cosmos-sdk/codec"

	"github.com/forbole/callisto/v4/types"
)

// SaveFeeGrantAllowance allows to store the fee grant allowances for the given block height
func (db *Db) SaveFeeGrantAllowance(allowance types.FeeGrant) error {
	// Store the accounts
	var accounts []types.Account
	accounts = append(accounts, types.NewAccount(allowance.Granter), types.NewAccount(allowance.Grantee))
	err := db.SaveAccounts(accounts)
	if err != nil {
		return fmt.Errorf("error while storing fee grant allowance accounts: %s", err)
	}

	stmt := `
INSERT INTO fee_grant_allowance(grantee_address, granter_address, allowance, height)
VALUES ($1, $2, $3, $4)
ON CONFLICT ON CONSTRAINT unique_fee_grant_allowance DO UPDATE
    SET allowance = excluded.allowance,
        height = excluded.height
WHERE fee_grant_allowance.height <= excluded.height`

	allowanceJSON, err := codec.ProtoMarshalJSON(allowance.Allowance, nil)
	if err != nil {
		return fmt.Errorf("error while marshaling grant allowance: %s", err)
	}

	_, err = db.SQL.Exec(stmt, allowance.Grantee, allowance.Granter, allowanceJSON, allowance.Height)
	if err != nil {
		return fmt.Errorf("error while saving fee grant allowance: %s", err)
	}

	return nil
}

// DeleteFeeGrantAllowance removes the fee grant allowance data from the database
func (db *Db) DeleteFeeGrantAllowance(allowance types.GrantRemoval) error {
	stmt := `DELETE FROM fee_grant_allowance WHERE grantee_address = $1 AND granter_address = $2 AND height <= $3`
	_, err := db.SQL.Exec(stmt, allowance.Grantee, allowance.Granter, allowance.Height)

	if err != nil {
		return fmt.Errorf("error while deleting grant allowance: %s", err)
	}
	return nil
}

// DeleteExpiredFeeGrantAllowances removes fee grant allowances whose expiration has passed as of
// blockTime. cosmos-sdk's own feegrant EndBlocker (RemoveExpiredAllowances) prunes expired
// allowances from the chain every block without emitting any event, so this can't rely on
// event-driven deletion the way an explicit revoke or a used-up allowance does.
//
// The allowance column already stores the full ProtoMarshalJSON of the allowance (BasicAllowance
// and PeriodicAllowance carry "expiration"/"basic.expiration" directly; AllowedMsgAllowance
// wraps one of those one level deeper under "allowance"), so the expiration is read straight out
// of that JSON instead of needing its own column.
func (db *Db) DeleteExpiredFeeGrantAllowances(blockTime time.Time) error {
	stmt := `
DELETE FROM fee_grant_allowance
WHERE COALESCE(
    allowance->>'expiration',
    allowance->'basic'->>'expiration',
    allowance->'allowance'->>'expiration',
    allowance->'allowance'->'basic'->>'expiration'
)::timestamptz <= $1`
	_, err := db.SQL.Exec(stmt, blockTime)
	if err != nil {
		return fmt.Errorf("error while deleting expired fee grant allowances: %s", err)
	}
	return nil
}
