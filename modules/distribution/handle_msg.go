package distribution

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/forbole/callisto/v4/types"
	juno "github.com/forbole/juno/v6/types"
	"github.com/rs/zerolog/log"
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
