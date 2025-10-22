package distribution

import (
	"fmt"

	juno "github.com/forbole/juno/v6/types"
	"github.com/rs/zerolog/log"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/forbole/callisto/v4/types"
	"github.com/forbole/callisto/v4/utils"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"


)

var msgFilter = map[string]bool{
	"/cosmos.distribution.v1beta1.MsgFundCommunityPool":           true,
	"/cosmos.distribution.v1beta1.MsgWithdrawValidatorCommission": true,
}

// HandleMsgExec implements modules.AuthzMessageModule
func (m *Module) HandleMsgExec(index int, _ int, executedMsg juno.Message, tx *juno.Transaction) error {
	return m.HandleMsg(index, executedMsg, tx)
}

// HandleMsg implements modules.MessageModule
func (m *Module) HandleMsg(_ int, msg juno.Message, tx *juno.Transaction) error {
	if _, ok := msgFilter[msg.GetType()]; !ok {
		return nil
	}

	log.Debug().Str("module", "distribution").Str("hash", tx.TxHash).Uint64("height", tx.Height).Msg(fmt.Sprintf("handling distribution message %s", msg.GetType()))

	if msg.GetType() == "/cosmos.distribution.v1beta1.MsgFundCommunityPool" {
		return m.updateCommunityPool(int64(tx.Height))
	}
	if msg.GetType() == "/cosmos.distribution.v1beta1.MsgWithdrawValidatorCommission" {
		cosmosMsg := utils.UnpackMessage(m.cdc, msg.GetBytes(), &distrtypes.MsgWithdrawValidatorCommission{})
		valAddr, err := sdk.ValAddressFromBech32(cosmosMsg.ValidatorAddress)
		if err != nil {
			return err
		}
		delegatorAddr := sdk.AccAddress(valAddr)
		var amount string

		for _, event := range tx.Events {
			if event.Type == "withdraw_commission" {
				// Get the amount attribute
				amountAttr, err := juno.FindAttributeByKey(event, sdk.AttributeKeyAmount)
				if err != nil {
					log.Warn().Str("module", "distribution").Str("hash", tx.TxHash).Msg("amount attribute not found in withdraw_commission event")
					continue
				}
				amount = amountAttr.Value
			}
		}

		fmt.Println("withdraw_commission, %s, %s, %d", delegatorAddr.String(), amount, tx.Height)
		
		return m.updateRewardEarned(delegatorAddr.String(), amount, int64(tx.Height))
	}
	return nil
}

func (m *Module) updateRewardEarned(delegator string, strCoin string, height int64) error {
	currentReward, _ := m.db.GetRewardEarnedByDelegator(delegator)
	rewardCoin, _ := sdk.ParseCoinNormalized(strCoin)

	if currentReward == nil {
		reward := types.NewRewardEarned(delegator, rewardCoin, height)
		fmt.Println("delegator %s first earned %s", delegator, rewardCoin.String())
		err := m.db.SaveRewardEarned(reward)
		if err != nil {
			return err
		}
	} else {
		rewardCoin = rewardCoin.Add(currentReward.Coin)
		reward := types.NewRewardEarned(delegator, rewardCoin, height)
		fmt.Println("delegator %s earned %s", delegator, rewardCoin.String())
		err := m.db.SaveRewardEarned(reward)
		if err != nil {
			return err
		}
	}
	return nil
}
