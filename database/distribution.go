package database

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"cosmossdk.io/math"
	dbtypes "github.com/forbole/callisto/v4/database/types"

	"github.com/forbole/callisto/v4/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lib/pq"
	"github.com/rs/zerolog/log"
)

// SaveCommunityPool allows to save for the given height the given total amount of coins
func (db *Db) SaveCommunityPool(coin sdk.DecCoins, height int64) error {
	query := `
INSERT INTO community_pool(coins, height) 
VALUES ($1, $2) 
ON CONFLICT (one_row_id) DO UPDATE 
    SET coins = excluded.coins,
        height = excluded.height
WHERE community_pool.height <= excluded.height`
	_, err := db.SQL.Exec(query, pq.Array(dbtypes.NewDbDecCoins(coin)), height)
	if err != nil {
		return fmt.Errorf("error while storing community pool: %s", err)
	}

	return nil
}

// -------------------------------------------------------------------------------------------------------------------

// SaveDistributionParams allows to store the given distribution parameters inside the database
func (db *Db) SaveDistributionParams(params *types.DistributionParams) error {
	paramsBz, err := json.Marshal(&params.Params)
	if err != nil {
		return fmt.Errorf("error while marshaling params: %s", err)
	}

	stmt := `
INSERT INTO distribution_params (params, height) 
VALUES ($1, $2)
ON CONFLICT (one_row_id) DO UPDATE 
    SET params = excluded.params,
      	height = excluded.height
WHERE distribution_params.height <= excluded.height`
	_, err = db.SQL.Exec(stmt, string(paramsBz), params.Height)
	if err != nil {
		return fmt.Errorf("error while storing distribution params: %s", err)
	}

	return nil
}

// -------------------------------------------------------------------------------------------------------------------

// SaveRewardEarned allows to save reward earned data for a delegator
func (db *Db) SaveRewardEarned(reward types.RewardEarned) error {
	query := `
INSERT INTO reward_earned (delegator_address, denom, amount, height)
VALUES ($1, $2, $3, $4)
ON CONFLICT (delegator_address) DO UPDATE
    SET denom = excluded.denom,
        amount = excluded.amount,
        height = excluded.height`

	_, err := db.SQL.Exec(query, reward.DelegatorAddress, reward.Coin.Denom, reward.Coin.Amount.String(), reward.Height)
	if err != nil {
		return fmt.Errorf("error while saving reward earned: %s", err)
	}

	return nil
}

// AddRewardEarned adds coin to delegatorAddress's running reward total for the given height.
// It's idempotent per delegator/height: if a contribution was already added at this height or a
// later one, the call is a no-op, so reprocessing an already-indexed block never double counts a
// transfer (the caller is expected to have already summed same-denom transfers to the same
// delegator within one block into a single call, so this only needs to guard against the block
// itself being reprocessed later).
//
// reward_earned holds a single denom per delegator (delegator_address is its whole primary key),
// so if delegatorAddress's stored total is already in a different denom than coin, the
// contribution is logged and skipped rather than corrupting the row or crashing - this can only
// happen if something other than a plain staking reward withdrawal (eg. a community pool spend
// grant) pays a former delegator in a different denom.
func (db *Db) AddRewardEarned(delegatorAddress string, coin sdk.Coin, height int64) error {
	var row dbtypes.RewardEarnedRow
	err := db.Sqlx.Get(&row, `SELECT delegator_address, denom, amount, height FROM reward_earned WHERE delegator_address = $1`, delegatorAddress)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("error while getting reward earned by delegator: %s", err)
	}

	if err == sql.ErrNoRows {
		return db.SaveRewardEarned(types.NewRewardEarned(delegatorAddress, coin, height))
	}

	if row.Height >= height {
		// Already recorded a contribution at this height or later - the block was reprocessed,
		// so this transfer is already part of the total.
		return nil
	}

	if row.Denom != coin.Denom {
		log.Warn().Str("module", "distribution").Str("delegator", delegatorAddress).
			Str("stored_denom", row.Denom).Str("new_denom", coin.Denom).
			Msg("skipping reward earned contribution in a different denom than what's already tracked")
		return nil
	}

	currentAmount, ok := math.NewIntFromString(row.Amount)
	if !ok {
		return fmt.Errorf("invalid amount format: %s", row.Amount)
	}

	newCoin := coin.Add(sdk.NewCoin(row.Denom, currentAmount))
	return db.SaveRewardEarned(types.NewRewardEarned(delegatorAddress, newCoin, height))
}

// GetRewardEarned retrieves reward earned data for a specific delegator, coin denom, and height
func (db *Db) GetRewardEarned(delegatorAddress, denom string, height int64) (*types.RewardEarned, error) {
	var row dbtypes.RewardEarnedRow
	query := `SELECT delegator_address, denom, amount, height FROM reward_earned WHERE delegator_address = $1 AND denom = $2 AND height = $3`

	err := db.Sqlx.Get(&row, query, delegatorAddress, denom, height)
	if err != nil {
		return nil, fmt.Errorf("error while getting reward earned: %s", err)
	}

	// Convert database row to types.RewardEarned
	amount, ok := math.NewIntFromString(row.Amount)
	if !ok {
		return nil, fmt.Errorf("invalid amount format: %s", row.Amount)
	}
	coin := sdk.NewCoin(row.Denom, amount)
	reward := types.NewRewardEarned(row.DelegatorAddress, coin, row.Height)

	return &reward, nil
}

// GetRewardEarnedByDelegator retrieves reward earned data for a specific delegator
func (db *Db) GetRewardEarnedByDelegator(delegatorAddress string) (*types.RewardEarned, error) {
	var row dbtypes.RewardEarnedRow
	query := `SELECT delegator_address, denom, amount, height FROM reward_earned WHERE delegator_address = $1`

	err := db.Sqlx.Get(&row, query, delegatorAddress)
	if err != nil {
		return nil, fmt.Errorf("error while getting reward earned by delegator: %s", err)
	}

	// Convert database row to types.RewardEarned
	amount, ok := math.NewIntFromString(row.Amount)
	if !ok {
		return nil, fmt.Errorf("invalid amount format: %s", row.Amount)
	}
	coin := sdk.NewCoin(row.Denom, amount)
	reward := types.NewRewardEarned(row.DelegatorAddress, coin, row.Height)

	return &reward, nil
}

