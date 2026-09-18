package distribution

import (
	juno "github.com/forbole/juno/v6/types"

	tmctypes "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/rs/zerolog/log"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

const (
	DistributionModuleAccount = "realio1jv65s3grqf6v6jl3dp4t6c9t9rk99cd8w2qk49"
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

	// Sum every reward transfer per delegator/denom across the whole block first, so that each
	// delegator/denom pair is only added to reward_earned once per height (AddRewardEarned relies
	// on that to safely skip a reprocessed block without also dropping a second same-block, same-
	// denom transfer to the same delegator, eg. from withdrawing from two validators at once).
	earnedByDelegator := make(map[string]sdk.Coins)
	for _, event := range events {
		if event.Type != banktypes.EventTypeTransfer {
			continue
		}

		sender, err := juno.FindAttributeByKey(event, banktypes.AttributeKeySender)
		if err != nil || sender.Value != DistributionModuleAccount {
			continue
		}

		delegator, err := juno.FindAttributeByKey(event, banktypes.AttributeKeyRecipient)
		if err != nil {
			continue
		}

		amount, err := juno.FindAttributeByKey(event, sdk.AttributeKeyAmount)
		if err != nil {
			continue
		}

		coin, err := sdk.ParseCoinNormalized(amount.Value)
		if err != nil {
			log.Error().Str("module", "distribution").Err(err).Str("amount", amount.Value).
				Msg("skipping reward earned event with unparseable amount")
			continue
		}

		earnedByDelegator[delegator.Value] = earnedByDelegator[delegator.Value].Add(coin)
	}

	for delegator, coins := range earnedByDelegator {
		for _, coin := range coins {
			err := m.db.AddRewardEarned(delegator, coin, block.Block.Height)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
