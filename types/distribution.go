package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
)

// DistributionParams represents the parameters of the x/distribution module
type DistributionParams struct {
	distrtypes.Params
	Height int64
}

// NewDistributionParams allows to build a new DistributionParams instance
func NewDistributionParams(params distrtypes.Params, height int64) *DistributionParams {
	return &DistributionParams{
		Params: params,
		Height: height,
	}
}

// -------------------------------------------------------------------------------------------------------------------

// RewardEarned represents a reward earned by a delegator
type RewardEarned struct {
	DelegatorAddress string
	Coin             sdk.Coin
	Height           int64
}

// NewRewardEarned allows to build a new RewardEarned instance
func NewRewardEarned(delegatorAddress string, coin sdk.Coin, height int64) RewardEarned {
	return RewardEarned{
		DelegatorAddress: delegatorAddress,
		Coin:             coin,
		Height:           height,
	}
}
