package database

import (
	"fmt"

	bridgetypes "github.com/realiotech/realio-network/x/bridge/types"
)

func (db *Db) SaveBridgeIn(hash string, msg *bridgetypes.MsgBridgeIn) error {
	query := `INSERT INTO bridge_in (hash, amount, denom, receiver) VALUES ($1, $2, $3, $4) ON CONFLICT (hash) DO NOTHING`

	_, err := db.SQL.Exec(query, hash, msg.Coin.Amount.String(), msg.Coin.Denom, msg.Receiver)
	if err != nil {
		return fmt.Errorf("error while storing bridge msg: %s", err)
	}

	return nil
}

func (db *Db) SaveBridgeOut(hash string, msg *bridgetypes.MsgBridgeOut) error {
	query := `INSERT INTO bridge_in (hash, amount, denom, sender) VALUES ($1, $2, $3, $4) ON CONFLICT (hash) DO NOTHING`

	_, err := db.SQL.Exec(query, hash, msg.Coin.Amount.String(), msg.Coin.Denom, msg.Signer)
	if err != nil {
		return fmt.Errorf("error while storing bridge msg: %s", err)
	}

	return nil
}

func (db *Db) SaveRates(height int64, rates []bridgetypes.DenomAndRateLimit) error {
	if len(rates) == 0 {
		return nil
	}

	query := `INSERT INTO rate_limit (denom, rate_limit, inflow, height) VALUES`

	var param []interface{}
	i := 0
	for _, rate := range rates {
		vi := i * 4
		query += fmt.Sprintf("($%d,$%d,$%d,$%d),", vi+1, vi+2, vi+3, vi+4)
		param = append(param, rate.Denom, rate.RateLimit.Ratelimit.String(), rate.RateLimit.CurrentInflow.String(), height)
		i++
	}

	query = query[:len(query)-1] // Remove trailing ","
	query += `
ON CONFLICT (denom, height) DO NOTHING`

	_, err := db.SQL.Exec(query, param...)
	if err != nil {
		return fmt.Errorf("error while saving rate limit: %s", err)
	}

	return nil
}