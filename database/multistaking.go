package database

import (
	"fmt"

	cosmossdk_io_math "cosmossdk.io/math"
	dbtypes "github.com/forbole/callisto/v4/database/types"
	"github.com/forbole/callisto/v4/types"

	multistakingtypes "github.com/realio-tech/multi-staking-module/x/multi-staking/types"
)

func (db *Db) SaveMultiStakingLocks(height int64, multiStakingLocks []*multistakingtypes.MultiStakingLock) error {
	query := "DELETE FROM ms_locks"
	_, err := db.SQL.Exec(query)
	if err != nil {
		return fmt.Errorf("error while deleting ms_unlocks: %s", err)
	}

	if len(multiStakingLocks) == 0 {
		return nil
	}

	query = `INSERT INTO ms_locks (staker_addr, val_addr, denom, amount, bond_weight, height) VALUES`

	var param []interface{}

	for i, msLock := range multiStakingLocks {
		vi := i * 6
		query += fmt.Sprintf("($%d,$%d,$%d,$%d,$%d,$%d),", vi+1, vi+2, vi+3, vi+4, vi+5, vi+6)
		mStakerAddr := msLock.LockID.MultiStakerAddr
		valAddr := msLock.LockID.ValAddr
		param = append(param, mStakerAddr, valAddr,
			msLock.LockedCoin.Denom, msLock.LockedCoin.Amount.String(), msLock.LockedCoin.BondWeight.String(), height)
	}

	query = query[:len(query)-1] // Remove trailing ","
	query += `
ON CONFLICT (staker_addr, val_addr) DO UPDATE 
	SET amount = excluded.amount,
		bond_weight = excluded.bond_weight,
		height = excluded.height
WHERE ms_locks.height <= excluded.height`

	_, err = db.SQL.Exec(query, param...)
	if err != nil {
		return fmt.Errorf("error while saving msLock: %s", err)
	}

	return nil
}

func (db *Db) SaveMultiStakingUnlocks(height int64, multiStakingUnlocks []*multistakingtypes.MultiStakingUnlock) error {

	query := "DELETE FROM ms_unlocks"
	_, err := db.SQL.Exec(query)
	if err != nil {
		return fmt.Errorf("error while deleting ms_unlocks: %s", err)
	}

	if len(multiStakingUnlocks) == 0 {
		return nil
	}

	query = `INSERT INTO ms_unlocks (staker_addr, val_addr, creation_height, denom, amount, bond_weight, height) VALUES`

	var param []interface{}
	count := 0
	for _, msUnlock := range multiStakingUnlocks {
		entries := msUnlock.Entries
		for _, entry := range entries {
			vi := count * 7
			query += fmt.Sprintf("($%d,$%d,$%d,$%d,$%d,$%d,$%d),", vi+1, vi+2, vi+3, vi+4, vi+5, vi+6, vi+7)
			mStakerAddr := msUnlock.UnlockID.MultiStakerAddr
			valAddr := msUnlock.UnlockID.ValAddr
			param = append(param, mStakerAddr, valAddr, entry.CreationHeight,
				entry.UnlockingCoin.Denom, entry.UnlockingCoin.Amount.String(), entry.UnlockingCoin.BondWeight.String(), height)
			count++
		}
	}

	query = query[:len(query)-1] // Remove trailing ","
	query += `
ON CONFLICT (staker_addr, val_addr, creation_height) DO UPDATE 
	SET amount = excluded.amount,
		bond_weight = excluded.bond_weight,
		height = excluded.height
WHERE ms_unlocks.height <= excluded.height`

	_, err = db.SQL.Exec(query, param...)
	if err != nil {
		return fmt.Errorf("error while saving msUnlock: %s", err)
	}

	return nil
}

func (db *Db) SaveUnbondingToken(height int64, multiStakingUnlocks []*multistakingtypes.MultiStakingUnlock) error {
	total := make(map[string]cosmossdk_io_math.Int)

	for _, msUnlock := range multiStakingUnlocks {
		entries := msUnlock.Entries
		for _, entry := range entries {
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

	if len(total) == 0 {
		return nil
	}

	query := `INSERT INTO token_unbonding (denom, amount, height) VALUES`

	var param []interface{}

	i := 0
	for denom, amount := range total {
		vi := i * 3
		query += fmt.Sprintf("($%d,$%d,$%d),", vi+1, vi+2, vi+3)

		param = append(param, denom, amount.String(), height)
		i++
	}

	query = query[:len(query)-1] // Remove trailing ","
	query += `
ON CONFLICT (denom) DO UPDATE 
	SET amount = excluded.amount,
		height = excluded.height
WHERE token_unbonding.height <= excluded.height`

	_, err := db.SQL.Exec(query, param...)
	if err != nil {
		return fmt.Errorf("error while saving token_unbonding: %s", err)
	}

	return nil
}

func (db *Db) SaveUnbondingToken2(height int64, total map[string]cosmossdk_io_math.Int) error {
	if len(total) == 0 {
		return nil
	}

	query := `INSERT INTO token_unbonding (denom, amount, height) VALUES`

	var param []interface{}

	i := 0
	for denom, amount := range total {
		vi := i * 3
		query += fmt.Sprintf("($%d,$%d,$%d),", vi+1, vi+2, vi+3)

		param = append(param, denom, amount.String(), height)
		i++
	}

	query = query[:len(query)-1] // Remove trailing ","
	query += `
ON CONFLICT (denom) DO UPDATE 
	SET amount = excluded.amount,
		height = excluded.height
WHERE token_unbonding.height <= excluded.height`

	_, err := db.SQL.Exec(query, param...)
	if err != nil {
		return fmt.Errorf("error while saving token_unbonding: %s", err)
	}

	return nil
}

func (db *Db) SaveBondedToken(height int64, multiStakingLocks []*multistakingtypes.MultiStakingLock) error {
	total := make(map[string]cosmossdk_io_math.Int)

	for _, msLock := range multiStakingLocks {
		denom := msLock.LockedCoin.Denom
		amount := msLock.LockedCoin.Amount
		value, exists := total[denom]
		if !exists {
			total[denom] = amount
		} else {
			total[denom] = value.Add(amount)
		}
	}

	if len(total) == 0 {
		return nil
	}

	query := `INSERT INTO token_bonded (denom, amount, height) VALUES`

	var param []interface{}

	i := 0
	for denom, amount := range total {
		vi := i * 3
		query += fmt.Sprintf("($%d,$%d,$%d),", vi+1, vi+2, vi+3)

		param = append(param, denom, amount.String(), height)
		i++
	}

	query = query[:len(query)-1] // Remove trailing ","
	query += `
ON CONFLICT (denom) DO UPDATE 
	SET amount = excluded.amount,
		height = excluded.height
WHERE token_bonded.height <= excluded.height`

	_, err := db.SQL.Exec(query, param...)
	if err != nil {
		return fmt.Errorf("error while saving token_bonded: %s", err)
	}

	return nil
}

func (db *Db) SaveBondedToken2(height int64, total map[string]cosmossdk_io_math.Int) error {
	if len(total) == 0 {
		return nil
	}

	query := `INSERT INTO token_bonded (denom, amount, height) VALUES`

	var param []interface{}

	i := 0
	for denom, amount := range total {
		vi := i * 3
		query += fmt.Sprintf("($%d,$%d,$%d),", vi+1, vi+2, vi+3)

		param = append(param, denom, amount.String(), height)
		i++
	}

	query = query[:len(query)-1] // Remove trailing ","
	query += `
ON CONFLICT (denom) DO UPDATE 
	SET amount = excluded.amount,
		height = excluded.height
WHERE token_bonded.height <= excluded.height`

	_, err := db.SQL.Exec(query, param...)
	if err != nil {
		return fmt.Errorf("error while saving token_bonded: %s", err)
	}

	return nil
}

func (db *Db) SaveValidatorDenom(height int64, validatorInfo []types.MSValidatorInfo) error {
	if len(validatorInfo) == 0 {
		return nil
	}

	query := `INSERT INTO validator_denom (val_addr, denom, height) VALUES`

	var param []interface{}
	for i, info := range validatorInfo {
		vi := i * 3
		query += fmt.Sprintf("($%d,$%d,$%d),", vi+1, vi+2, vi+3)
		param = append(param, info.ConsensusAddress, info.Denom, height)
	}

	query = query[:len(query)-1] // Remove trailing ","
	query += `
ON CONFLICT (val_addr) DO UPDATE 
	SET denom = excluded.denom,
    	height = excluded.height
WHERE validator_denom.height <= excluded.height`

	_, err := db.SQL.Exec(query, param...)
	if err != nil {
		return fmt.Errorf("error while saving ValidatorDenom: %s", err)
	}

	return nil
}

func (db *Db) SaveMSEvent(msEvents []dbtypes.MSEvent, height int64) error {
	if len(msEvents) == 0 {
		return nil
	}

	query := `INSERT INTO ms_event (height, name, val_addr, del_addr, amount) VALUES`

	var param []interface{}
	for i, msEvent := range msEvents {
		vi := i * 5
		query += fmt.Sprintf("($%d,$%d,$%d,$%d,$%d),", vi+1, vi+2, vi+3, vi+4, vi+5)
		param = append(param, height, msEvent.Name, msEvent.ValAddr, msEvent.DelAddr, msEvent.Amount)
	}

	query = query[:len(query)-1] // Remove trailing ","

	_, err := db.SQL.Exec(query, param...)
	if err != nil {
		return fmt.Errorf("error while saving msEvents: %s", err)
	}

	return nil
}

func (db *Db) SaveMultiStakingLock(height int64, entry *multistakingtypes.MultiStakingLock) error {

	query := `INSERT INTO ms_locks (staker_addr, val_addr, denom, amount, bond_weight, height) VALUES ($1,$2,$3,$4,$5,$6)`

	var param []interface{}
	param = append(param, entry.LockID.MultiStakerAddr, entry.LockID.ValAddr,
		entry.LockedCoin.Denom, entry.LockedCoin.Amount.String(), entry.LockedCoin.BondWeight.String(), height)

	query += `
ON CONFLICT (staker_addr, val_addr) DO UPDATE 
	SET amount = excluded.amount,
		bond_weight = excluded.bond_weight,
		height = excluded.height
WHERE ms_locks.height <= excluded.height`

	_, err := db.SQL.Exec(query, param...)
	if err != nil {
		return fmt.Errorf("error while saving msLock: %s", err)
	}

	return nil
}

func (db *Db) SaveMultiStakingUnlock(height int64, unlock *multistakingtypes.MultiStakingUnlock) error {
	query := `INSERT INTO ms_unlocks (staker_addr, val_addr, creation_height, denom, amount, bond_weight, height) VALUES`

	var param []interface{}
	count := 0
	entries := unlock.Entries
	for _, entry := range entries {
		vi := count * 7
		query += fmt.Sprintf("($%d,$%d,$%d,$%d,$%d,$%d,$%d),", vi+1, vi+2, vi+3, vi+4, vi+5, vi+6, vi+7)
		mStakerAddr := unlock.UnlockID.MultiStakerAddr
		valAddr := unlock.UnlockID.ValAddr
		param = append(param, mStakerAddr, valAddr, entry.CreationHeight,
			entry.UnlockingCoin.Denom, entry.UnlockingCoin.Amount.String(), entry.UnlockingCoin.BondWeight.String(), height)
		count++
	}

	query = query[:len(query)-1] // Remove trailing ","
	query += `
ON CONFLICT (staker_addr, val_addr, creation_height) DO UPDATE 
	SET amount = excluded.amount,
		bond_weight = excluded.bond_weight,
		height = excluded.height
WHERE ms_unlocks.height <= excluded.height`

	_, err := db.SQL.Exec(query, param...)
	if err != nil {
		return fmt.Errorf("error while saving msUnlock: %s", err)
	}

	return nil
}

func (db *Db) DropMultiStakingUnlock(stakerAddr, valAddr string) error {
	query := "DELETE from ms_unlocks where staker_addr = $1 AND val_addr = $2"
	_, err := db.SQL.Exec(query, stakerAddr, valAddr)
	if err != nil {
		return fmt.Errorf("error while saving msLock: %s", err)
	}

	return nil
}
