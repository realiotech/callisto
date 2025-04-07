CREATE TABLE etransaction(
    ehash TEXT,
    transaction_hash TEXT,
    partition_id BIGINT NOT NULL DEFAULT 0,
    FOREIGN KEY (transaction_hash, partition_id) REFERENCES transaction (hash, partition_id),
	CONSTRAINT unique_ehash_per_tx UNIQUE (ehash, partition_id)
)PARTITION BY LIST(partition_id);