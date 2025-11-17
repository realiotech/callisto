package database

import (
	"encoding/json"
	"fmt"

	"cosmossdk.io/math"
	dbtypes "github.com/forbole/callisto/v4/database/types"

	"github.com/forbole/callisto/v4/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lib/pq"
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

