package distribution

import (
	juno "github.com/forbole/juno/v6/types"

	tmctypes "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/rs/zerolog/log"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
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
			err := m.updateRewardEarned(delegator.Value, amount.Value, block.Block.Height)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
