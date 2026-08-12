### Changed

- Transaction feed messages now report the PGA round in which the transaction was sequenced in `pga_round`, previously a placeholder. The field is omitted when zero (DelayedMessages, FIFO ordering).
