### Changed

- Transaction feed messages now report the PGA round in which the transaction was sequenced in `pga_round`, previously a placeholder. The field is omitted when zero, i.e. for txs not ordered by PGA (delayed messages, FIFO ordering, internal and resequenced txs).
