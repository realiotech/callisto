package distribution

import (
	"fmt"

	juno "github.com/forbole/juno/v6/types"

	tmctypes "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/rs/zerolog/log"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/forbole/callisto/v4/types"
)

// HandleBlock implements BlockModule
func (m *Module) HandleBlock(
	block *tmctypes.ResultBlock, res *tmctypes.ResultBlockResults, _ []*juno.Transaction, _ *tmctypes.ResultValidators,
) error {
	log.Debug().Str("module", "distribution").Int64("height", block.Block.Height).
		Msg("updating reward by event")
	events := res.FinalizeBlockEvents
	for _, tx := range res.TxsResults {
		events = append(events, tx.Events...)
	}

	for _, event := range events {
		// Delegation rewards
		if event.Type == "withdraw_rewards" {
			delegator, _ := juno.FindAttributeByKey(event, stakingtypes.AttributeKeyDelegator)
			amount, _ := juno.FindAttributeByKey(event, sdk.AttributeKeyAmount)
			log.Debug().Str("module", "distribution").Int64("height", block.Block.Height).
				Msg("updating txs by event")

			currentReward, _ := m.db.GetRewardEarnedByDelegator(delegator.Value)
			rewardCoin, _ := sdk.ParseCoinNormalized(amount.Value)

			if currentReward == nil {
				reward := types.NewRewardEarned(delegator.Value, rewardCoin, block.Block.Height)
				fmt.Println("delegator %s first earned %s", delegator.Value, rewardCoin.String())
				err := m.db.SaveRewardEarned(reward)
				if err != nil {
					return err
				}
			} else {
				rewardCoin = rewardCoin.Add(currentReward.Coin)
				reward := types.NewRewardEarned(delegator.Value, rewardCoin, block.Block.Height)
				fmt.Println("delegator %s earned %s", delegator.Value, rewardCoin.String())
				err := m.db.SaveRewardEarned(reward)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}
