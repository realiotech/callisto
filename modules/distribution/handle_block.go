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

	for _, event := range events {
		if event.Type == banktypes.EventTypeTransfer {
			sender, err := juno.FindAttributeByKey(event, banktypes.AttributeKeySender)
			if err == nil && sender.Value == DistributionModuleAccount {
				delegator, _ := juno.FindAttributeByKey(event, banktypes.AttributeKeyRecipient)
				amount, _ := juno.FindAttributeByKey(event, sdk.AttributeKeyAmount)
				err := m.updateRewardEarned(delegator.Value, amount.Value, block.Block.Height)
				if err != nil {
					return err
				}
			}

		}

	}

	return nil
}
