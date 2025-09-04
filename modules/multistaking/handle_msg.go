package multistaking

import (
	"fmt"

	cosmossdk_io_math "cosmossdk.io/math"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/rs/zerolog/log"

	juno "github.com/forbole/juno/v6/types"

	dbtypes "github.com/forbole/callisto/v4/database/types"
	"github.com/forbole/callisto/v4/utils"
	multistakingtypes "github.com/realio-tech/multi-staking-module/x/multi-staking/types"
)

var msgFilter = map[string]bool{
	"/cosmos.staking.v1beta1.MsgCreateValidator": true,
	"/cosmos.staking.v1beta1.MsgEditValidator":   true,
	"/cosmos.staking.v1beta1.MsgDelegate":        true,
	"/cosmos.staking.v1beta1.MsgUndelegate":      true,
	"/cosmos.staking.v1beta1.MsgBeginRedelegate": true,
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

	case "/cosmos.staking.v1beta1.MsgDelegate":
		cosmosMsg := utils.UnpackMessage(m.cdc, msg.GetBytes(), &stakingtypes.MsgDelegate{})
		return m.UpdateLockAndUnlockInfo(int64(tx.Height), cosmosMsg.DelegatorAddress, cosmosMsg.ValidatorAddress)

	case "/cosmos.staking.v1beta1.MsgBeginRedelegate":
		cosmosMsg := utils.UnpackMessage(m.cdc, msg.GetBytes(), &stakingtypes.MsgBeginRedelegate{})
		err := m.UpdateLockAndUnlockInfo(int64(tx.Height), cosmosMsg.DelegatorAddress, cosmosMsg.ValidatorDstAddress)
		if err != nil {
			return err
		}

		return m.UpdateLockAndUnlockInfo(int64(tx.Height), cosmosMsg.DelegatorAddress, cosmosMsg.ValidatorSrcAddress)

	case "/cosmos.staking.v1beta1.MsgUndelegate":
		cosmosMsg := utils.UnpackMessage(m.cdc, msg.GetBytes(), &stakingtypes.MsgUndelegate{})
		return m.UpdateLockAndUnlockInfo(int64(tx.Height), cosmosMsg.DelegatorAddress, cosmosMsg.ValidatorAddress)

	case "/cosmos.staking.v1beta1.MsgCancelUnbondingDelegation":
		cosmosMsg := utils.UnpackMessage(m.cdc, msg.GetBytes(), &stakingtypes.MsgCancelUnbondingDelegation{})
		return m.UpdateLockAndUnlockInfo(int64(tx.Height), cosmosMsg.DelegatorAddress, cosmosMsg.ValidatorAddress)
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

	msunlock, err := m.source.GetMultiStakingUnlock(height, stakerAddr, valAddr)
	if err != nil {
		return err
	}

	// Update token totals BEFORE saving the new lock/unlock data
	// This ensures we read the old data from the database before it's overwritten
	err = m.UpdateLockToken(height, stakerAddr, valAddr, mslock)
	if err != nil {
		return err
	}

	err = m.UpdateUnlockToken(height, stakerAddr, valAddr, msunlock)
	if err != nil {
		return err
	}

	// Now save the new lock/unlock data
	if mslock != nil {
		err = m.db.SaveMultiStakingLock(height, mslock)
		if err != nil {
			return err
		}
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
			return fmt.Errorf("NewIntFromString false", tokenRow.Amount)
		}

		total[denom] = amount
	}

	for _, lockRow := range lockRows {
		denom := lockRow.Denom
		amount, ok := cosmossdk_io_math.NewIntFromString(lockRow.Amount)

		if !ok {
			return fmt.Errorf("NewIntFromString false", lockRow.Amount)
		}
		amount = amount.Neg()

		value, exists := total[denom]
		if !exists {
			total[denom] = amount
		} else {
			total[denom] = value.Add(amount)
		}
	}

	// Add new lock amount to totals (if lock exists)
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
			return fmt.Errorf("NewIntFromString false", tokenRow.Amount)
		}

		total[denom] = amount
	}

	for _, row := range unlockRows {
		denom := row.Denom
		amount, ok := cosmossdk_io_math.NewIntFromString(row.Amount)
		if !ok {
			return fmt.Errorf("NewIntFromString false", row.Amount)
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
