package multistaking

import (
	"fmt"

	cosmossdk_io_math "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/rs/zerolog/log"

	juno "github.com/forbole/juno/v6/types"

	abci "github.com/cometbft/cometbft/abci/types"
	dbtypes "github.com/forbole/callisto/v4/database/types"
	"github.com/forbole/callisto/v4/utils"
	multistakingtypes "github.com/realio-tech/multi-staking-module/x/multi-staking/types"
)

var msgFilter = map[string]bool{
	"/cosmos.staking.v1beta1.MsgCreateValidator":    true,
	"/cosmos.staking.v1beta1.MsgEditValidator":      true,
	"/cosmos.staking.v1beta1.MsgDelegate":           true,
	"/cosmos.staking.v1beta1.MsgUndelegate":         true,
	"/cosmos.staking.v1beta1.MsgBeginRedelegate":    true,
	"/cosmos.evm.vm.v1.MsgEthereumTx":               true,
	"/multistaking.v1.MsgDelegateEVM":               true,
	"/multistaking.v1.MsgUndelegateEVM":             true,
	"/multistaking.v1.CancelUnbondingEVMDelegation": true,
	"/multistaking.v1.MsgCreateEVMValidator":        true,
	"/multistaking.v1.MsgBeginRedelegateEVM":        true,
}

// HandleMsgExec implements modules.AuthzMessageModule
func (m *Module) HandleMsgExec(index int, _ int, executedMsg juno.Message, tx *juno.Transaction) error {
	return m.HandleMsg(index, executedMsg, tx)
}

// HandleMsg implements MessageModule
func (m *Module) HandleMsg(_ int, msg juno.Message, tx *juno.Transaction) error {
	if _, ok := msgFilter[msg.GetType()]; !ok {
		return nil
	}

	log.Debug().Str("module", "staking").Str("hash", tx.TxHash).Uint64("height", tx.Height).Msg(fmt.Sprintf("handling staking message %s", msg.GetType()))

	switch msg.GetType() {
	case "/cosmos.staking.v1beta1.MsgCreateValidator":
		cosmosMsg := utils.UnpackMessage(m.cdc, msg.GetBytes(), &stakingtypes.MsgCreateValidator{})
		return m.UpdateLockAndUnlockInfo(int64(tx.Height), cosmosMsg.DelegatorAddress, cosmosMsg.ValidatorAddress)
	case "/multistaking.v1.MsgCreateEVMValidator":
		cosmosMsg := utils.UnpackMessage(m.cdc, msg.GetBytes(), &multistakingtypes.MsgCreateEVMValidator{})
		return m.UpdateLockAndUnlockInfo(int64(tx.Height), cosmosMsg.DelegatorAddress, cosmosMsg.ValidatorAddress)

	case "/cosmos.staking.v1beta1.MsgDelegate":
		cosmosMsg := utils.UnpackMessage(m.cdc, msg.GetBytes(), &stakingtypes.MsgDelegate{})
		return m.UpdateLockAndUnlockInfo(int64(tx.Height), cosmosMsg.DelegatorAddress, cosmosMsg.ValidatorAddress)
	case "/multistaking.v1.MsgDelegateEVM":
		cosmosMsg := utils.UnpackMessage(m.cdc, msg.GetBytes(), &multistakingtypes.MsgDelegateEVM{})
		return m.UpdateLockAndUnlockInfo(int64(tx.Height), cosmosMsg.DelegatorAddress, cosmosMsg.ValidatorAddress)

	case "/cosmos.staking.v1beta1.MsgBeginRedelegate":
		cosmosMsg := utils.UnpackMessage(m.cdc, msg.GetBytes(), &stakingtypes.MsgBeginRedelegate{})
		err := m.UpdateLockAndUnlockInfo(int64(tx.Height), cosmosMsg.DelegatorAddress, cosmosMsg.ValidatorDstAddress)
		if err != nil {
			return err
		}

		return m.UpdateLockAndUnlockInfo(int64(tx.Height), cosmosMsg.DelegatorAddress, cosmosMsg.ValidatorSrcAddress)
	case "/multistaking.v1.MsgBeginRedelegateEVM":
		cosmosMsg := utils.UnpackMessage(m.cdc, msg.GetBytes(), &multistakingtypes.MsgBeginRedelegateEVM{})
		err := m.UpdateLockAndUnlockInfo(int64(tx.Height), cosmosMsg.DelegatorAddress, cosmosMsg.ValidatorDstAddress)
		if err != nil {
			return err
		}

		return m.UpdateLockAndUnlockInfo(int64(tx.Height), cosmosMsg.DelegatorAddress, cosmosMsg.ValidatorSrcAddress)

	case "/cosmos.staking.v1beta1.MsgUndelegate":
		cosmosMsg := utils.UnpackMessage(m.cdc, msg.GetBytes(), &stakingtypes.MsgUndelegate{})
		return m.UpdateLockAndUnlockInfo(int64(tx.Height), cosmosMsg.DelegatorAddress, cosmosMsg.ValidatorAddress)
	case "/multistaking.v1.MsgUndelegateEVM":
		cosmosMsg := utils.UnpackMessage(m.cdc, msg.GetBytes(), &multistakingtypes.MsgUndelegateEVM{})
		return m.UpdateLockAndUnlockInfo(int64(tx.Height), cosmosMsg.DelegatorAddress, cosmosMsg.ValidatorAddress)

	case "/cosmos.staking.v1beta1.MsgCancelUnbondingDelegation":
		cosmosMsg := utils.UnpackMessage(m.cdc, msg.GetBytes(), &stakingtypes.MsgCancelUnbondingDelegation{})
		return m.CompleteUnbonding(int64(tx.Height), cosmosMsg.DelegatorAddress, cosmosMsg.ValidatorAddress)
	case "/multistaking.v1.CancelUnbondingEVMDelegation":
		cosmosMsg := utils.UnpackMessage(m.cdc, msg.GetBytes(), &multistakingtypes.MsgCancelUnbondingEVMDelegation{})
		return m.CompleteUnbonding(int64(tx.Height), cosmosMsg.DelegatorAddress, cosmosMsg.ValidatorAddress)

	case "/cosmos.evm.vm.v1.MsgEthereumTx":
		// For evm tx, handle by event
		events := tx.Events
		var delegator abci.EventAttribute
		EventLoop:for _, event := range events {
			switch event.Type {
			// Use for Redelegate case, it missing delegator address
			// So we get it from withdraw_rewards event
			case "withdraw_rewards":
				delegator, _ = juno.FindAttributeByKey(event, stakingtypes.AttributeKeyDelegator)
			case stakingtypes.EventTypeCreateValidator:
				valAddr, _ := juno.FindAttributeByKey(event, stakingtypes.AttributeKeyValidator)
				valAcc, _ := sdk.ValAddressFromBech32(valAddr.Value)
				delAddr := sdk.AccAddress(valAcc)
				m.UpdateLockAndUnlockInfo(int64(tx.Height), delAddr.String(), valAddr.Value)
				// break loop to avoid duplicate events
				break EventLoop

			case stakingtypes.EventTypeDelegate:
				valAddr, _ := juno.FindAttributeByKey(event, stakingtypes.AttributeKeyValidator)
				delAddr, _ := juno.FindAttributeByKey(event, stakingtypes.AttributeKeyDelegator)

				m.UpdateLockAndUnlockInfo(int64(tx.Height), delAddr.Value, valAddr.Value)
				// break loop to avoid duplicate events
				break EventLoop

			case stakingtypes.EventTypeUnbond:
				valAddr, _ := juno.FindAttributeByKey(event, stakingtypes.AttributeKeyValidator)
				delAddr, _ := juno.FindAttributeByKey(event, stakingtypes.AttributeKeyDelegator)
				m.UpdateLockAndUnlockInfo(int64(tx.Height), delAddr.Value, valAddr.Value)
				// break loop to avoid duplicate events
				break EventLoop

			case stakingtypes.EventTypeCancelUnbondingDelegation:
				valAddr, _ := juno.FindAttributeByKey(event, stakingtypes.AttributeKeyValidator)
				delAddr, _ := juno.FindAttributeByKey(event, stakingtypes.AttributeKeyDelegator)
				
				m.CompleteUnbonding(int64(tx.Height), delAddr.Value, valAddr.Value)
				// break loop to avoid duplicate events
				break EventLoop

			case stakingtypes.EventTypeRedelegate:
				valAddr1, _ := juno.FindAttributeByKey(event, stakingtypes.AttributeKeySrcValidator)
				valAddr2, _ := juno.FindAttributeByKey(event, stakingtypes.AttributeKeyDstValidator)
				m.UpdateLockAndUnlockInfo(int64(tx.Height), delegator.Value, valAddr1.Value)
				m.UpdateLockAndUnlockInfo(int64(tx.Height), delegator.Value, valAddr2.Value)
				// break loop to avoid duplicate events
				break EventLoop

			}
		}

	}

	return nil
}

func (m *Module) UpdateLockAndUnlockInfo(height int64, stakerAddr string, valAddr string) error {
	log.Trace().Str("module", "multistaking").Str("operation", "lock info").
		Msg("updating lock and unlock info")

	mslock, err := m.source.GetMultiStakingLock(height, stakerAddr, valAddr)
	if err != nil {
		return err
	}

	err = m.UpdateLockToken(height, stakerAddr, valAddr, mslock)
	if err != nil {
		return err
	}

	if mslock != nil {
		err = m.db.SaveMultiStakingLock(height, mslock)
		if err != nil {
			return err
		}
	}

	msunlock, err := m.source.GetMultiStakingUnlock(height, stakerAddr, valAddr)
	if err != nil {
		return err
	}

	err = m.UpdateUnlockToken(height, stakerAddr, valAddr, msunlock)
	if err != nil {
		return err
	}

	if msunlock != nil {
		err = m.db.SaveMultiStakingUnlock(height, msunlock)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *Module) UpdateLockToken(height int64, stakerAddr string, valAddr string, lock *multistakingtypes.MultiStakingLock) error {
	log.Trace().Str("module", "multistaking").Str("operation", "lock info").
		Msg("updating lock and unlock info")

	var lockRows []dbtypes.LockRow
	err := m.db.SQL.Select(&lockRows, `SELECT * FROM ms_locks WHERE staker_addr = $1 AND val_addr = $2`, stakerAddr, valAddr)
	if err != nil {
		return err
	}

	var tokenRows []dbtypes.MSTokenRow
	err = m.db.SQL.Select(&tokenRows, `SELECT * FROM token_bonded`)
	if err != nil {
		return err
	}

	total := make(map[string]cosmossdk_io_math.Int)
	for _, tokenRow := range tokenRows {
		denom := tokenRow.Denom
		amount, ok := cosmossdk_io_math.NewIntFromString(tokenRow.Amount)
		if !ok {
			return fmt.Errorf("NewIntFromString failed for amount %q", tokenRow.Amount)
		}

		total[denom] = amount
	}

	for _, lockRow := range lockRows {
		denom := lockRow.Denom
		amount, ok := cosmossdk_io_math.NewIntFromString(lockRow.Amount)

		if !ok {
			return fmt.Errorf("NewIntFromString failed for amount %q", lockRow.Amount)
		}
		amount = amount.Neg()

		value, exists := total[denom]
		if !exists {
			total[denom] = amount
		} else {
			total[denom] = value.Add(amount)
		}
	}

	if lock != nil {
		denom := lock.LockedCoin.Denom
		value, exists := total[denom]
		if !exists {
			total[denom] = lock.LockedCoin.Amount
		} else {
			total[denom] = value.Add(lock.LockedCoin.Amount)
		}
	}

	return m.db.SaveBondedToken2(height, total)
}

func (m *Module) UpdateUnlockToken(height int64, stakerAddr string, valAddr string, unlock *multistakingtypes.MultiStakingUnlock) error {
	log.Trace().Str("module", "multistaking").Str("operation", "lock info").
		Msg("updating lock and unlock info")

	var unlockRows []dbtypes.UnlockRow
	err := m.db.SQL.Select(&unlockRows, `SELECT * FROM ms_unlocks WHERE staker_addr = $1 AND val_addr = $2`, stakerAddr, valAddr)
	if err != nil {
		return err
	}

	var tokenRows []dbtypes.MSTokenRow
	err = m.db.SQL.Select(&tokenRows, `SELECT * FROM token_unbonding`)
	if err != nil {
		return err
	}

	total := make(map[string]cosmossdk_io_math.Int)
	for _, tokenRow := range tokenRows {
		denom := tokenRow.Denom
		amount, ok := cosmossdk_io_math.NewIntFromString(tokenRow.Amount)
		if !ok {
			return fmt.Errorf("NewIntFromString failed for amount %q", tokenRow.Amount)
		}

		total[denom] = amount
	}

	for _, row := range unlockRows {
		denom := row.Denom
		amount, ok := cosmossdk_io_math.NewIntFromString(row.Amount)
		if !ok {
			return fmt.Errorf("NewIntFromString failed for amount %q", row.Amount)
		}
		amount = amount.Neg()

		value, exists := total[denom]
		if !exists {
			total[denom] = amount
		} else {
			total[denom] = value.Add(amount)
		}
	}
	if unlock != nil {
		for _, entry := range unlock.Entries {
			denom := entry.UnlockingCoin.Denom
			amount := entry.UnlockingCoin.Amount
			value, exists := total[denom]
			if !exists {
				total[denom] = amount
			} else {
				total[denom] = value.Add(amount)
			}
		}
	}
	return m.db.SaveUnbondingToken2(height, total)
}
