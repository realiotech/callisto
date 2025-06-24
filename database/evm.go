package database

import (
	"fmt"

	"github.com/forbole/juno/v6/types/config"
)

func (db *Db) SaveEvmTx(height int64, txHash string, eHash string) error {
	var partitionID int64
	partitionSize := config.Cfg.Database.PartitionSize
	if partitionSize > 0 {
		partitionID = height / partitionSize
		err := db.CreatePartitionIfNotExists("etransaction", partitionID)
		if err != nil {
			return err
		}
	}

	return db.saveEvmTxInsidePartition(txHash, eHash, partitionID)
}

func (db *Db) saveEvmTxInsidePartition(hash string, eHash string, partitionID int64) error {
	query := `INSERT INTO etransaction (ehash, transaction_hash, partition_id)
		VALUES ($1, $2, $3)`

	_, err := db.SQL.Exec(query, eHash, hash, partitionID)
	if err != nil {
		return fmt.Errorf("error while storing evm msg: %s", err)
	}

	return nil
}
