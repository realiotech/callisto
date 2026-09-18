package database

import (
	"fmt"

	dbtypes "github.com/forbole/callisto/v4/database/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/forbole/callisto/v4/types"
	"github.com/lib/pq"
)

// SaveSupply allows to save for the given height the given total amount of coins
func (db *Db) SaveSupply(coins sdk.Coins, height int64) error {
	query := `
INSERT INTO supply (coins, height) 
VALUES ($1, $2) 
ON CONFLICT (one_row_id) DO UPDATE 
    SET coins = excluded.coins,
    	height = excluded.height
WHERE supply.height <= excluded.height`

	_, err := db.SQL.Exec(query, pq.Array(dbtypes.NewDbCoins(coins)), height)
	if err != nil {
		return fmt.Errorf("error while storing supply: %s", err)
	}

	return nil
}

func (db *Db) SaveTokenHolder(tokens map[string]int, height int64) error {
	if len(tokens) == 0 {
		return nil
	}

	query := `INSERT INTO token_holder (denom, num_holder, height) VALUES`

	var param []interface{}
	i := 0
	for denom, amount := range tokens {
		vi := i * 3
		query += fmt.Sprintf("($%d,$%d,$%d),", vi+1, vi+2, vi+3)
		param = append(param, denom, amount, height)
		i++
	}

	query = query[:len(query)-1] // Remove trailing ","
	query += `
ON CONFLICT (denom) DO UPDATE 
	SET num_holder = excluded.num_holder,
    	height = excluded.height
WHERE token_holder.height <= excluded.height`

	_, err := db.SQL.Exec(query, param...)
	if err != nil {
		return fmt.Errorf("error while saving token_holder: %s", err)
	}

	return nil
}

// SaveAccountBalances replaces the entire balance table with the given set of balances.
// It's meant to be used only when balances represents the complete current holder set
// (eg. rebuilt from all denom owners at a given height), since any address/denom pair
// that is not present in balances will be removed.
func (db *Db) SaveAccountBalances(balances []types.AccountBalance, height int64) error {
	_, err := db.SQL.Exec(`DELETE FROM balance`)
	if err != nil {
		return fmt.Errorf("error while deleting balance: %s", err)
	}

	if len(balances) == 0 {
		return nil
	}

	query := `INSERT INTO balance (address, amount, denom, height) VALUES`

	var param []interface{}
	for i, balance := range balances {
		vi := i * 4
		query += fmt.Sprintf("($%d,$%d,$%d,$%d),", vi+1, vi+2, vi+3, vi+4)
		param = append(param, balance.Address, balance.Amount, balance.Denom, height)
	}

	query = query[:len(query)-1] // Remove trailing ","
	query += `
ON CONFLICT (address, denom) DO UPDATE
	SET amount = excluded.amount,
    	height = excluded.height
WHERE balance.height <= excluded.height`

	_, err = db.SQL.Exec(query, param...)
	if err != nil {
		return fmt.Errorf("error while saving AccountBalances: %s", err)
	}

	return nil
}

// UpdateAccountBalances refreshes the balances of the given addresses, replacing whatever
// is currently stored for them with balances. Any address/denom pair that used to be
// stored for one of addresses but is not present in balances (eg. because it dropped to
// zero and the chain no longer reports it) is removed, instead of being left stale.
func (db *Db) UpdateAccountBalances(addresses []string, balances []types.AccountBalance, height int64) error {
	if len(addresses) == 0 {
		return nil
	}

	_, err := db.SQL.Exec(`DELETE FROM balance WHERE address = ANY($1)`, pq.Array(addresses))
	if err != nil {
		return fmt.Errorf("error while deleting stale AccountBalances: %s", err)
	}

	if len(balances) == 0 {
		return nil
	}

	query := `INSERT INTO balance (address, amount, denom, height) VALUES`

	var param []interface{}
	for i, balance := range balances {
		vi := i * 4
		query += fmt.Sprintf("($%d,$%d,$%d,$%d),", vi+1, vi+2, vi+3, vi+4)
		param = append(param, balance.Address, balance.Amount, balance.Denom, height)
	}

	query = query[:len(query)-1] // Remove trailing ","
	query += `
ON CONFLICT (address, denom) DO UPDATE
	SET amount = excluded.amount,
    	height = excluded.height
WHERE balance.height <= excluded.height`

	_, err = db.SQL.Exec(query, param...)
	if err != nil {
		return fmt.Errorf("error while saving AccountBalances: %s", err)
	}

	return nil
}
