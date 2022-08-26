# Migrating State and History from a Classic Arbitrum Node to Nitro

Nitro software has the option to initialize a chain with state imported from a classic arbitrum node. Rinkeby Testnet, and later Arbitrum One, use this option. Entire state of the rollup is preserved when it is upgraded from classic to nitro.

When importing history - Nitro's genesis block will be the block that follows the imported history, and not block 0.

The recommended way to initialize a nitro node is to point it to a pre-initialized database using the `--init.url` option. This documentation is for users who wish to create the full state on their own classic node.

## Exporting Data from a Classic Arbitrum Node

Launch node with the option `--node.rpc.nitroexport.enable=true` note: this is only recommended for nodes with no public/external interfaces. All exported data will be written to directory "nitroexport" under the classic instance directory - e.g. `${HOME}/.arbitrum/rinkeby/nitroexport`.
Make sure the classic node has read the entire rollup state. 

**Important Note:** Exporting the state on your own classic node should produce the same state as using files supplied by offchain labs (e.g. the same genesis blockhash). However, multiple exports of the same state will not necessarily create identical intermediate files. For example - state export is done in parallel, so order of entries in the file is not deterministic.

### Exporting Block & Transaction History

These are block-headers, transactions and receipts executed in the classic node. Nitro node uses the history to be able to answer simple requests, like eth_getTransactionReceipt, from the classic history. The last block in the chain is the only one that affects the genesis block: timestamp is copied from the last block, and parentHash is taken from the last block's blockHash.

- RPC call `arb_exportHistory` with parameter `"latest"` will initiate history export. It will return immediately.
- `arb_exportHistoryStatus` will return the latest block exported, or an error if export failed.
- Data will be stored in dir `nitroexport/nitro/l2chaindata/ancient`. 

### Exporting Outbox Messages

This data does not impact consensus and is optional. It allows a nitro node to provide the information required when redeeming a withdrawal made on the classic rollup.

- RPC call `arb_exportOutbox` with parameter `"0xffffffffffffffff"` will initiate block export. It will return immediately.
- `arb_exportOutboxStatus` will return the latest outbox batch exported, or an error is export failed.
- Data will be stored in dir `nitroexport/nitro/classic-msg`.


### Exporting State

Rollup state is exported as a series of json files. State read from these json files will be added to nitro's genesis block.

- RPC call `arb_exportState` with parameter `"latest"` will initiate stet export. Unless disconnected - this will only return after state export is done.
- Data will be created in dir `nitroexport/state/<block_number>/`.


## Running Nitro Node Initialization

State Import requires more resources than normal run of a nitro node.

- Place l2chaindata and classic-msg (optional) directories in nitro's instance directory - e.g. ${HOME}/.arbitrum/rinkeby-nitro/
- Launch the node with argument `--init.import-file=/path/to/state/index.json`

### Other Nitro Options
- `--init.accounts-per-sync` allows the node to make partial database writes to hard-disk during initialization, allowing memory to be freed. This should be used if memory load is very high. A reasonable initial value to try would be 100000. Systems with constrained memory might require a lower value.
- `--init.then-quit` causes the node to quit after initialization is done.
- `--init.force` for an already-initialized node, forces the node to recalculate nitro's genesis block. If the genesis blockhash does not match what's in the database - the node will panic.
