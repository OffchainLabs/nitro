# Run a batchposter draft

# Regular setup

Some of the default configuration flags and values are

| flag | default value | description |
| --- | --- | --- |
| `--node.batch-poster.enable` | `false` | Enable posting batches to L1 |
| `--node.batch-poster.max-size` | 100000 | Default value is overwritten if its an L3 (90000) |
| `--node.batch-poster.poll-interval` | 10 seconds | how long to wait after no batches are ready to be posted before checking again |
| `--node.batch-poster.error-delay` | 10 seconds | how long to delay after error posting batch |
| `--node.batch-poster.compression-level` | 11 | batch compression level (Arbitrum uses brotli compression which has compression levels from 1-11) |
| `--node.batch-poster.parent-chain-wallet.private-key` | none | Sets the private key of the parent chains wallet  |

if you created an L3 arbitrum chain and generated your node config file for a full node. then the only values for the batch poster that will bet set are `--node.batch-poster.enable` = True, `--node.batch-poster.max-size` = 90000, and `--node.batch-poster.parent-chain-wallet.private-key` = `YOUR_PRIVATE_KEY`

Overall if you are the chain owner. relying on the Arbitrum Chain SDK to create your config file is the best option.

To add a new batch poster, call the `setIsBatchPoster(address,bool)` method of the `SequencerInbox` contract on the parent chain:

```tsx
cast send --rpc-url $PARENT_CHAIN_RPC --private-key $OWNER_PRIVATE_KEY $SEQUENCER_INBOX_ADDRESS "setIsBatchPoster(address,bool) ()" $NEW_BATCH_POSTER_ADDRESS true
```

# Be part of the high availability sequencer

(This content is still pending)

# Queued tx db selection

Queued txs are transactions that the sequencer has ordered and are ready for execution.

- noop: Configured with `--node.batch-poster.data-poster.use-noop-storage`
- redis: Configured with `--node.batch-poster.redis-url`
- DB: Configured with `--node.batch-poster.data-poster.use-db-storage`

Only one database option can be chosen, so you can not have both a redis server and a DB being utilized at the same time.

Note: if your parent chain is an arbitrum chain or doesn’t have a mempool then this section can be ignored as noop storage will be automatically chosen (This is because batches are post sequentially, and will only post a new batch if the previous transaction went through, therefore no database tracking queued transactions is required)

### Noop

Noop storage does not store any queuedTxs, This is beneficial when the parent chain is an Arbitrum chain or one without a mempool since the sequencer takes in every transaction immediately and is never in mempool limbo due to low gas. 

When noop is enabled. The batch poster will wait for a confirmation that the transaction has gone through. If the transaction reverted then the batch poster will just try again without having to halt operation. 

There is no RBF logic since that is for chains where a mempool is used

### DB

DB will store data locally in the node and allows for persistent `queuedTXs`. Only a single batch poster can use the DB. the DB allows for Replace By Fee so transactions will not get stuck in the parent chains mempool. If a transaction is reverted then the batch poster will have to halt.

### Redis

Redis is a fast local memory database ran separately from the nitro software, meaning that node restarts will keep `queuedTXs`.

Using Redis allows for multiple batch posters to run in parallel and read from it due to redisLock. which is useful for redundancy and high availability. 

Storing transactions also allows for RBF which can stop transactions from being stuck in the mempool without enough gas. However using redis means the batch poster will halt if a transaction reverts

to set up redis you also set up a `redis-signer` value with the flag `--node.batch-poster-data_poster.redis-signer.signing-key` 

The default values are DB on by default, and Noop on by default if and only if the parent chain does not have a mempool (Any L3 who’s parent is an arbitrum chain)

Key differences 

| **Feature** | **Noop** | **DB** | **Redis** |
| --- | --- | --- | --- |
| **Persistence** | None | Local disk | External Redis |
| **Survives restarts** | No | Yes | Yes |
| **Replace-by-fee** | No | Yes | Yes |
| **Multiple posters** | No | No | Yes |
| **Tolerates reverts** | Yes | No | No |
| **Waits for receipts** | Yes | No | No |

# Enable Blob Posting

This subsection explains how to configure an arbitrum node to post [EIP-4844](https://eips.ethereum.org/EIPS/eip-4844) blob transaction to the parent chain, which can significantly reduce data availability costs

### Prerequisites

Before enabling blob transactions, verify that your setup meets these requirements

1. Chain configuration
    - Your arbitrum chain must be running in **Rollup mode**
2. Parent chain compatibility
    
    Your parent chain (typically Ethereum mainnet or a testnet) must support [EIP-4844](https://eips.ethereum.org/EIPS/eip-4844). You can verify this by checking that recent block headers contain:
    
    - `ExcessBlobGas` field
    - `BlobGasUsed` field
3. ArbOS version
    
    Your ArbOS version must be 20 or higher. To check your current version:
    
    **Method 1: Smart contract call** 
    
    Call the `arbOSVersion()` function on the ArbSys precompile contract:
    
    - Contract address: `0x0000000000000000000000000000000000000064`
    - Function: `arbOSVersion()` returns `uint256`
    - You can call this using any Ethereum client or block explorer on your Arbitrum chain
    
    **Method 2: Using `cast` (if you have Foundry installed)**
    
    ```solidity
    cast call 0x0000000000000000000000000000000000000064 "arbOSVersion()" --rpc-url YOUR_ARBITRUM_RPC_URL
    ```
    
    If your version is below 20, upgrade by following the [**ArbOS upgrade guide**](https://docs.arbitrum.io/launch-arbitrum-chain/configure-your-chain/common-configurations/arbos-upgrade).
    
    ### Configuration
    
    To enable blob transaction posting, add the following configuration to your node:
    
    ```json
    {
      "node": {
        "batch-poster": {
          "post-4844-blobs": true
        }
      }
    }
    ```
    
    After updating your configuration:
    
    1. Save the configuration file
    2. Restart your Arbitrum node
    3. Monitor the logs to confirm blob posting is active
    
    ### Verification
    
    Once restarted, you can verify that blob transactions are being posted successfully by monitoring your node logs.
    
    ### Log messages to look for
    
    When a blob transaction is successfully posted, you'll see a log entry similar to:
    
    ```verilog
    INFO [05-23|00:49:16.160] BatchPoster: batch sent  sequenceNumber=6 from=24 
    to=28 prevDelayed=13 currentDelayed=14 totalSegments=9
    numBlobs=1
    ```
    
    **Key indicator**: The `numBlobs` field shows the number of blobs included in the transaction:
    
    - `numBlobs=0`: Traditional calldata transaction was posted
    - `numBlobs>0`: Blob transaction was successfully posted (in the example above, 1 blob was sent)
    
    ## Troubleshooting
    
    ### **Why is my node still posting calldata instead of blobs?**
    
    Your node may continue using calldata in these scenarios:
    
    1. **Cost optimization**: When blob gas prices are high, calldata posting may be more economical, but you can set the `-node.batch-poster.ignore-blob-price` flag to `true` to force the batch poster to use blobs.
    2. **Batch Type Switching Protection**: After a non-blob transaction is posted, the next 16 transactions will also use calldata to prevent frequent switching
    
    Check your node logs for blob-related error messages and verify that your parent chain is accessible and fully synced.
    
    ## Optional parameters
    
    You can also set the following optional parameters to control blob posting behavior: 
    
    | **Flag** | **Description** |
    | --- | --- |
    | `--node.batch-poster.ignore-blob-price` | Boolean. Default: `false`. If the parent chain supports `EIP-4844` blobs and `ignore-blob-price` is set to `true`, the batch poster will use `EIP-4844` blobs even if using calldata is cheaper. Can be `true` or `false`. |
    | `--parent-chain.blob-client.authorization` | String. Default: `""`. Value to send with the HTTP Authorization: header for Beacon REST requests, must include both scheme and scheme parameters |
    | `--parent-chain.blob-client.secondary-beacon-url` | String. Default: `""`. Value to send with the HTTP Authorization: header for Beacon REST requests, must include both scheme and scheme parameters |
    | `--node.batch-poster.data-poster.blob-tx-replacement-times` | durationSlice. Default: `[5m0s,10m0s,30m0s,1h0m0s,4h0m0s,8h0m0s,16h0m0s,22h0m0s]`. comma-separated list of durations since first posting a blob transaction to attempt a replace-by-fee |
    | `--node.batch-poster.data-poster.max-blob-tx-tip-cap-gwei` | float. Default: `1`. the maximum tip cap to post `EIP-4844` blob-carrying transactions at |
    | `--node.batch-poster.data-poster.min-blob-tx-tip-cap-gwei` | float. Default: `1`. the minimum tip cap to post `EIP-4844` blob-carrying transactions at |
    

# Batchposter Revenue Config

To change revenue configurations for a batch poster, the first thing that should be done is checking the list of current registered batch posters through the [**`ArbAggregator`](https://docs.arbitrum.io/build-decentralized-apps/precompiles/reference#arbaggregator)** precompile by calling `getBatchPosters() (address[])` 

```verilog
cast call --rpc-url $ORBIT_CHAIN_RPC 0x000000000000000000000000000000000000006D "getBatchPosters() (address[])"
```

While there are other ways to get the list of batch posters for an arbitrum chain, this method only lists batch posters who are registered and have posted at least a single batch, which is better for revenue reasons.

Once you have the batch poster address you can obtain the fee collector address for that batch poster using the `getFeeCollector(address)(address)` from the [**`ArbAggregator`](https://docs.arbitrum.io/build-decentralized-apps/precompiles/reference#arbaggregator) precompile**

```verilog
cast call --rpc-url $ORBIT_CHAIN_RPC 0x000000000000000000000000000000000000006D "getFeeCollector(address) (address)" $BATCH_POSTER_ADDRESS
```

this can also be done in the Arbitrum Chain SDK

```tsx
const orbitChainClient = createPublicClient({
    chain: <OrbitChainDefinition>,
    transport: http(),
}).extend(arbAggregatorActions);

const networkFeeAccount = await orbitChainClient.arbAggregatorReadContract({
    functionName: 'getFeeCollector',
    args: [<BatchPosterAddress>],
});
```

Note: Before setting a fee collector for a batch poster you must make sure that the batch poster is registered in the `BatchPostersTable`. This can be achieved by

- Manually calling `ArbAggregator.addBatchPoster()` for the address, or
- The address having successfully posted at least one batch

To set a new fee collector for a specific batch poster, use the method `setFeeCollector(address, address)` of the [**`ArbAggregator`**](https://docs.arbitrum.io/build-decentralized-apps/precompiles/reference#arbaggregator) precompile:

```tsx
cast send --rpc-url $ORBIT_CHAIN_RPC --private-key $OWNER_PRIVATE_KEY 0x000000000000000000000000000000000000006D "setFeeCollector(address,address) ()" $BATCH_POSTER_ADDRESS $NEW_FEECOLLECTOR_ADDRESS
```

This can also be done in the Arbitrum Chain SDK

```tsx
const owner = privateKeyToAccount(<OwnerPrivateKey>);
const orbitChainClient = createPublicClient({
    chain: <OrbitChainDefinition>,
    transport: http(),
}).extend(arbAggregatorActions);

const transactionRequest = await orbitChainClient.arbAggregatorPrepareTransactionRequest({
    functionName: 'setFeeCollector',
    args: [<BatchPosterAddress>, <NewFeeCollectorAddress>],
    upgradeExecutor: false,
    account: owner.address,
});

await orbitChainClient.sendRawTransaction({
    serializedTransaction: await owner.signTransaction(transactionRequest),
});
```

To add a new batch poster, call the `setIsBatchPoster(address,bool)` method of the `SequencerInbox` contract on the parent chain:

```tsx
cast send --rpc-url $PARENT_CHAIN_RPC --private-key $OWNER_PRIVATE_KEY $SEQUENCER_INBOX_ADDRESS "setIsBatchPoster(address,bool) ()" $NEW_BATCH_POSTER_ADDRESS true
```

## Setting revenue values

Note: These values are only editable by the chain owners 

There are on chain values in ArbOS which Set how much ArbOS charges per L1 gas spent on transaction data. This can be set by communicating with the [`ArbOwner`](https://docs.arbitrum.io/build-decentralized-apps/precompiles/reference#arbowner) Precompile

```tsx
cast send --rpc-url $ARB_CHAIN_RPC --private-key $OWNER_PRIVATE_KEY $0x0000000000000000000000000000000000000070 "setL1PricingRewardRate(uint64) ()" NEW_L1_PRICING_REWARD
```

Along with setPerBatchGasCharge() which Sets the base charge (in L1 gas) attributed to each data batch in the calldata pricer. This can be called with

```tsx
cast send --rpc-url ARB_CHAIN_RPC --private-key OWNER_PRIVATE_KEY 0x0000000000000000000000000000000000000070 "setPerBatchGasCharge(int64) ()" NEW_BATCH_GAS_CHARGE
```

# Batchposter interval config

The batch poster has a minimum frequency which is primarily set by `—-node.batch-poster.max-delay` parameter in the [nitro node configuration](https://github.com/OffchainLabs/nitro/blob/master/arbnode/batch_poster.go) (set via the JSON config file or command-line flags when deploying an Orbit chain). It defines the maximum amount of time the batch poster will wait after receiving a transaction before posting a batch that includes it. The default value is 1 hour (3600 seconds)

- **Configuration options:** In the `node.batch-poster` section of the config, e.g., `"max-delay": "30m"` for a 30-minute maximum wait. Lower values will increase batch posting frequency, but at the cost of potential smaller and less efficient batches at times of low activity which increases the gas cost on the parent chain. If there are no transactions in a batch than this setting does not apply
- **Prevention of Issues:** Shorter max delay reduces the opportunity for transaction reordering in the sequencer due to waiting for shorter periods of time. It also limits exposure to chain reorgs, since batches post sooner which anchor them to the L1 before potential fluctuations can invalidate sequencing. Extremely low posting time is not recommended as spamming the L1 with batches, increases cost while lacking benefits
- **Recommended Settings:** For high-throughput chains, set to 5-15 minutes to balance latency and efficiency. For low activity chains, keep the default value of one hour.

The batch poster also has `--node.batch-poster.max-size` parameter represented in bytes. It is the maximum size a batch can be. If the total queued transactions compression estimate is greater than the max size, then the batch poster will post max size amount of transactions to the L1. The default value is 100000 bytes

Lower values will result in increased frequency of batch posting during high activity. And due to brotli compression, smaller batch files almost always lead to less optimal compression compared to larger files. which means gas price overall will cost more over 2 smaller batches compared to 1 larger batch, assuming the 2 smaller batches contain the same transactions as the 1 larger batch. Calldata and blob posting has an upper limit so raising this value too high can cause issues, while lowering the value can cause inefficient compression and batch spamming on high activity chains, which leaves the recommended value to be 100,000 bytes