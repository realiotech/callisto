package multistaking

import (
	"fmt"

	abci "github.com/cometbft/cometbft/abci/types"

	juno "github.com/forbole/juno/v6/types"

	tmctypes "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/rs/zerolog/log"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	dbtypes "github.com/forbole/callisto/v4/database/types"
)

// HandleBlock implements BlockModule
func (m *Module) HandleBlock(
	block *tmctypes.ResultBlock, res *tmctypes.ResultBlockResults, _ []*juno.Transaction, _ *tmctypes.ResultValidators,
) error {
	events := res.FinalizeBlockEvents
	for _, tx := range res.TxsResults {
		events = append(events, tx.Events...)
	}

	err := m.updateTxsByEvent(block.Block.Height, events)
	if err != nil {
		fmt.Printf("Error when updateTxsByEvent, error: %s", err)
	}
	return nil
}

func (m *Module) updateTxsByEvent(height int64, events []abci.Event) error {
	log.Debug().Str("module", "multistaking").Int64("height", height).
		Msg("updating txs by event")

	var msEvents []dbtypes.MSEvent
	var delegator abci.EventAttribute
	for _, event := range events {
		switch event.Type {
		// Use for Redelegate case, it missing delegator address
		// So we get it from withdraw_rewards event
		case "withdraw_rewards":
			delegator, _ = juno.FindAttributeByKey(event, stakingtypes.AttributeKeyDelegator)
		case stakingtypes.EventTypeCreateValidator:
			valAddr, _ := juno.FindAttributeByKey(event, stakingtypes.AttributeKeyValidator)
			valAcc, _ := sdk.ValAddressFromBech32(valAddr.Value)
			delAddr := sdk.AccAddress(valAcc)
			m.UpdateLockAndUnlockInfo(height, delAddr.String(), valAddr.Value)

		case stakingtypes.EventTypeDelegate:
			valAddr, _ := juno.FindAttributeByKey(event, stakingtypes.AttributeKeyValidator)
			delAddr, _ := juno.FindAttributeByKey(event, stakingtypes.AttributeKeyDelegator)
			amount, _ := juno.FindAttributeByKey(event, sdk.AttributeKeyAmount)
			msEvent, err := dbtypes.NewMSEvent("delegate", valAddr.Value, delAddr.Value, amount.Value)

			if err == nil {
				msEvents = append(msEvents, msEvent)
			}
			m.UpdateLockAndUnlockInfo(height, delAddr.Value, valAddr.Value)

		case stakingtypes.EventTypeUnbond:
			valAddr, _ := juno.FindAttributeByKey(event, stakingtypes.AttributeKeyValidator)
			delAddr, _ := juno.FindAttributeByKey(event, stakingtypes.AttributeKeyDelegator)
			amount, _ := juno.FindAttributeByKey(event, sdk.AttributeKeyAmount)
			msEvent, err := dbtypes.NewMSEvent("unbond", valAddr.Value, delAddr.Value, amount.Value)

			if err == nil {
				msEvents = append(msEvents, msEvent)
			}
			m.UpdateLockAndUnlockInfo(height, delAddr.Value, valAddr.Value)

		case stakingtypes.EventTypeCancelUnbondingDelegation:
			valAddr, _ := juno.FindAttributeByKey(event, stakingtypes.AttributeKeyValidator)
			delAddr, _ := juno.FindAttributeByKey(event, stakingtypes.AttributeKeyDelegator)
			amount, _ := juno.FindAttributeByKey(event, sdk.AttributeKeyAmount)
			msEvent, err := dbtypes.NewMSEvent("cancel_unbond", valAddr.Value, delAddr.Value, amount.Value)

			if err == nil {
				msEvents = append(msEvents, msEvent)
			}
			m.UpdateLockAndUnlockInfo(height, delAddr.Value, valAddr.Value)

		case stakingtypes.EventTypeCompleteRedelegation:
			valAddr1, _ := juno.FindAttributeByKey(event, stakingtypes.AttributeKeySrcValidator)
			valAddr2, _ := juno.FindAttributeByKey(event, stakingtypes.AttributeKeyDstValidator)
			delAddr, _ := juno.FindAttributeByKey(event, stakingtypes.AttributeKeyDelegator)
			m.UpdateLockAndUnlockInfo(height, delAddr.Value, valAddr1.Value)
			m.UpdateLockAndUnlockInfo(height, delAddr.Value, valAddr2.Value)

		case stakingtypes.EventTypeRedelegate:
			valAddr1, _ := juno.FindAttributeByKey(event, stakingtypes.AttributeKeySrcValidator)
			valAddr2, _ := juno.FindAttributeByKey(event, stakingtypes.AttributeKeyDstValidator)
			m.UpdateLockAndUnlockInfo(height, delegator.Value, valAddr1.Value)
			m.UpdateLockAndUnlockInfo(height, delegator.Value, valAddr2.Value)

		case stakingtypes.EventTypeCompleteUnbonding:
			valAddr, _ := juno.FindAttributeByKey(event, stakingtypes.AttributeKeyValidator)
			delAddr, _ := juno.FindAttributeByKey(event, stakingtypes.AttributeKeyDelegator)
			m.CompleteUnbonding(height, delAddr.Value, valAddr.Value)
		}
	}

	if len(msEvents) == 0 {
		return nil
	}
	return m.db.SaveMSEvent(msEvents, height)
}
