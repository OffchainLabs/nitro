# Partner Technical Q&A

This document answers technical questions from our partner.

---

## Question 1: RPC, Logging and Database Verification

### 1-1) How to check sync status via eth_syncing RPC call?

**Example using curl:**

```bash
# Check sync status
curl -X POST -H "Content-Type: application/json" \
  --data '{"jsonrpc":"2.0","method":"eth_syncing","id":1}' \
  http://localhost:8547

# If fully synced, returns:
{"jsonrpc":"2.0","id":1,"result":false}

# If still syncing, returns something like:
{
  "jsonrpc":"2.0",
  "id":1,
  "result":{
    "batchSeen": 12345,
    "batchProcessed": 12300,
    "messageOfProcessedBatch": 100000,
    "msgCount": 100500,
    "blockNum": 50000,
    "maxBlockNum": 50100,
    "feedPendingMessageCount": 50
  }
}
```

**Example using JavaScript:**

```javascript
const response = await fetch('http://localhost:8547', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    jsonrpc: '2.0',
    method: 'eth_syncing',
    id: 1
  })
});
const result = await response.json();

if (result.result === false) {
  console.log('Node is fully synced');
} else {
  console.log('Node is syncing:', result.result);
}
```

**Code location:** `execution/gethexec/sync_monitor.go:177` and `arbnode/sync_monitor.go:202`

---

### 1-2) How to enable trace-level logging?

**Method 1: Command line option**

```bash
# Start nitro node with trace level logging
./nitro --log-level=TRACE

# Or for DAS server
./daserver --log-level=TRACE

# Or for relay
./relay --log-level=TRACE
```

**Method 2: JSON config file**

```json
{
  "log-level": "TRACE"
}
```

Then run:
```bash
./nitro --conf.file /path/to/config.json
```

**Available log levels (from lowest to highest):**
- `TRACE` - most detailed
- `DEBUG`
- `INFO`
- `WARN`
- `ERROR`
- `CRIT` - least detailed

**Code location:** `cmd/genericconf/loglevel.go:15-38`

---

### 1-3) How to verify database consistency?

**Method 1: Compare block height with other nodes**

```bash
# Get latest block number from your node
curl -X POST -H "Content-Type: application/json" \
  --data '{"jsonrpc":"2.0","method":"eth_blockNumber","id":1}' \
  http://localhost:8547

# Compare with public Arbitrum endpoint or other trusted node
curl -X POST -H "Content-Type: application/json" \
  --data '{"jsonrpc":"2.0","method":"eth_blockNumber","id":1}' \
  https://arb1.arbitrum.io/rpc
```

**Method 2: Use dbconv tool for database verification**

```bash
# Full verification (keys and values)
./dbconv --verify=full --src.data=/path/to/database

# Keys only verification (faster)
./dbconv --verify=keys --src.data=/path/to/database
```

**Method 3: Compare block hash at specific height**

```bash
# Get block hash at specific height
curl -X POST -H "Content-Type: application/json" \
  --data '{"jsonrpc":"2.0","method":"eth_getBlockByNumber","params":["0x1000", false],"id":1}' \
  http://localhost:8547

# Compare the "hash" field with other nodes
```

---

## Question 2: batch-poster.reorg-resistance-margin option

### What does this option mean?

```
--node.batch-poster.reorg-resistance-margin=0
```

**Explanation:**

This option controls how long the batch poster waits before posting a batch, to protect against L1 reorgs.

- **Default value:** 10 minutes
- **Purpose:** Prevent posting batch if its first message is within this time margin from L1 minimum bounds
- **Requires:** `l1-block-bound` option NOT set to "ignore"

**What "margin" means here:**

The batch poster checks if the first message in a batch is too close to the current L1 block time/number. If the message is within the margin, the batch poster will wait before posting. This prevents the batch from being affected by potential L1 reorgs.

**Setting to 0:**

```bash
--node.batch-poster.reorg-resistance-margin=0
```

This **disables** the reorg resistance check. The batch poster will post batches immediately without waiting.

**When to use:**
- For testnets where reorg safety is not critical
- When you have full control of L1 and reorgs won't happen
- NOT recommended for production mainnet deployments

**Code location:** `arbnode/batch_poster.go:179` and `arbnode/batch_poster.go:1626-1644`

---

## Question 3: ArbOS Versions

### What does ArbOS do?

ArbOS is the **Layer 2 EVM hypervisor** that:
1. Manages the L2 execution environment
2. Accounts for and manages network resources (gas, fees)
3. Produces blocks from incoming messages
4. Operates the Geth instance for smart contract execution
5. Handles cross-chain communication (L1 <-> L2)
6. Manages retryable tickets

### ArbOS 40 changes

ArbOS 40 includes these main changes:

1. **EIP-2935 support (historical block hashes)**
   - Deploys HistoryStorage contract at `0x0000F90827F1C53a10cb7A02335B175320002935`
   - Buffer size: 393168 blocks (not standard 8191)
   - Uses L2 block number (not L1 block number)

2. **MaxWasmSize parameterization**
   - Before ArbOS 40: MaxWasmSize was constant
   - After ArbOS 40: Can be configured via `SetWasmMaxSize`
   - Initial value: 128 KB

3. **New precompile methods (available from ArbOS 40)**
   - `SetCalldataPriceIncrease`
   - `IsCalldataPriceIncreaseEnabled`
   - `SetWasmMaxSize`

### ArbOS versions and Nitro compatibility

| ArbOS Version | Features |
|---------------|----------|
| 2-11 | L1 pricing, fee management |
| 20 | Brotli compression level (0→1) |
| 30-32 | Stylus (WASM support) |
| 40-41 | EIP-2935, MaxWasmSize parameterization |
| 50-51 | DIA, multi-constraint fixes |
| 60 | Future |

**Current max supported version:** `ArbosVersion_51` (in latest Nitro)

**Version number allocation:**
- 12-19, 21-29, 33-39, 42-49: Reserved for Orbit chain custom upgrades
- Main versions (20, 30, 40, 50): Used by Nitro mainchain

**Code location:** `go-ethereum/params/config_arbitrum.go`

---

## Question 4: Split Validator Node Configuration

### How to split your current config into two nodes?

**Original config** runs sequencer + batch-poster + staker all in one node. You can split into:

1. **Sequencer Node** (handles sequencing, batch posting)
2. **Validator Node** (handles staking/validation only)

### Sequencer Node Config (node 1):

```json
{
  "chain": {
    "info-json": "[YOUR_CHAIN_INFO]",
    "name": "YOUR_CHAIN_NAME"
  },
  "parent-chain": {
    "connection": {
      "url": "https://YOUR_PARENT_CHAIN_RPC"
    }
  },
  "http": {
    "addr": "0.0.0.0",
    "port": 8459,
    "vhosts": ["*"],
    "corsdomain": ["*"],
    "api": ["eth", "net", "web3", "arb", "debug"]
  },
  "node": {
    "sequencer": true,
    "delayed-sequencer": {
      "enable": true,
      "use-merge-finality": false,
      "finalize-distance": 1
    },
    "batch-poster": {
      "max-size": 90000,
      "enable": true,
      "parent-chain-wallet": {
        "private-key": "YOUR_BATCH_POSTER_KEY"
      }
    },
    "feed": {
      "output": {
        "enable": true,
        "addr": "0.0.0.0",
        "port": 7765
      }
    },
    "staker": {
      "enable": false
    },
    "dangerous": {
      "no-sequencer-coordinator": true,
      "disable-blob-reader": true
    },
    "data-availability": {
      "enable": true,
      "sequencer-inbox-address": "YOUR_SEQUENCER_INBOX",
      "parent-chain-node-url": "https://YOUR_PARENT_CHAIN_RPC",
      "rest-aggregator": {
        "enable": true,
        "urls": ["http://YOUR_DAS_SERVER:9877"]
      },
      "rpc-aggregator": {
        "enable": true,
        "assumed-honest": 1,
        "backends": "[{\"url\":\"http://YOUR_DAS:9876\",\"pubkey\":\"YOUR_PUBKEY\",\"signermask\":1}]"
      }
    }
  },
  "execution": {
    "forwarding-target": "",
    "sequencer": {
      "enable": true,
      "max-tx-data-size": 85000,
      "max-block-speed": "250ms"
    },
    "caching": {
      "archive": true
    }
  }
}
```

### Validator/Staker Node Config (node 2):

```json
{
  "chain": {
    "info-json": "[YOUR_CHAIN_INFO]",
    "name": "YOUR_CHAIN_NAME"
  },
  "parent-chain": {
    "connection": {
      "url": "https://YOUR_PARENT_CHAIN_RPC"
    }
  },
  "http": {
    "addr": "0.0.0.0",
    "port": 8460,
    "vhosts": ["*"],
    "corsdomain": ["*"],
    "api": ["eth", "net", "web3", "arb"]
  },
  "node": {
    "sequencer": false,
    "staker": {
      "enable": true,
      "strategy": "MakeNodes",
      "parent-chain-wallet": {
        "private-key": "YOUR_STAKER_KEY"
      }
    },
    "feed": {
      "input": {
        "url": ["ws://SEQUENCER_NODE:7765"]
      }
    },
    "data-availability": {
      "enable": true,
      "sequencer-inbox-address": "YOUR_SEQUENCER_INBOX",
      "parent-chain-node-url": "https://YOUR_PARENT_CHAIN_RPC",
      "rest-aggregator": {
        "enable": true,
        "urls": ["http://YOUR_DAS_SERVER:9877"]
      }
    }
  },
  "execution": {
    "forwarding-target": "http://SEQUENCER_NODE:8459",
    "sequencer": {
      "enable": false
    }
  }
}
```

### About BOLD config for v3.7.4

For v3.7.4, BOLD support is available. You can add:

```json
{
  "node": {
    "bold": {
      "enable": true,
      "strategy": "MakeNodes"
    }
  }
}
```

**Note:** The "strategy" field should be lowercase "MakeNodes" or "Watchtower", not "makenodes".

**Full BOLD config options:**

```json
{
  "node": {
    "bold": {
      "enable": true,
      "assertion-posting-interval": "15m",
      "assertion-scanning-interval": "1m",
      "assertion-confirming-interval": "1m",
      "api": false,
      "api-host": "127.0.0.1",
      "api-port": 9393,
      "rpc-block-number": "finalized",
      "auto-deposit": true,
      "enable-fast-confirmation": false,
      "state-provider-config": {
        "validator-name": "my-validator",
        "check-batch-finality": true,
        "machine-leaves-cache-path": "machine-hashes-cache"
      }
    }
  }
}
```

---

## Question 5: SetL1PricePerUnit and SetL1PricingRewardRate

### How to call these functions?

Based on the example at `examples/setup-fee-distributor-contract/index.ts`, you need to change like this:

**Original code:**

```typescript
const setFeeCollectorTransactionRequest =
  await orbitChainPublicClient.arbOwnerPrepareTransactionRequest({
    functionName: 'setInfraFeeAccount',
    args: [rewardDistributorAddress],
    upgradeExecutor: tokenBridgeContracts.orbitChainContracts.upgradeExecutor,
    account: chainOwner.address,
  });
```

**Modified for SetL1PricePerUnit (set to 0):**

```typescript
// Call 1: Set L1 Price Per Unit to 0
const setL1PricePerUnitRequest =
  await orbitChainPublicClient.arbOwnerPrepareTransactionRequest({
    functionName: 'setL1PricePerUnit',
    args: [BigInt(0)],  // pricePerUnit = 0
    upgradeExecutor: tokenBridgeContracts.orbitChainContracts.upgradeExecutor,
    account: chainOwner.address,
  });

await orbitChainPublicClient.sendRawTransaction({
  serializedTransaction: await chainOwner.signTransaction(setL1PricePerUnitRequest),
});
```

**Modified for SetL1PricingRewardRate (set to 0):**

```typescript
// Call 2: Set L1 Pricing Reward Rate to 0
const setL1PricingRewardRateRequest =
  await orbitChainPublicClient.arbOwnerPrepareTransactionRequest({
    functionName: 'setL1PricingRewardRate',
    args: [BigInt(0)],  // perUnitReward = 0
    upgradeExecutor: tokenBridgeContracts.orbitChainContracts.upgradeExecutor,
    account: chainOwner.address,
  });

await orbitChainPublicClient.sendRawTransaction({
  serializedTransaction: await chainOwner.signTransaction(setL1PricingRewardRateRequest),
});
```

### Full example:

```typescript
import {
  createPublicClient,
  createWalletClient,
  http,
  defineChain,
} from 'viem';
import { privateKeyToAccount } from 'viem/accounts';
import {
  createRollupFetchCoreContracts,
  createTokenBridgeFetchTokenBridgeContracts,
  arbOwnerPublicActions,
} from '@arbitrum/orbit-sdk';
import { getParentChainFromId, sanitizePrivateKey } from '@arbitrum/orbit-sdk/utils';

const chainOwner = privateKeyToAccount(sanitizePrivateKey(process.env.CHAIN_OWNER_PRIVATE_KEY));

const parentChain = getParentChainFromId(Number(process.env.PARENT_CHAIN_ID));
const parentChainPublicClient = createPublicClient({
  chain: parentChain,
  transport: http(),
});

const orbitChain = defineChain({
  id: Number(process.env.ORBIT_CHAIN_ID),
  network: 'Orbit chain',
  name: 'orbit',
  nativeCurrency: { name: 'Ether', symbol: 'ETH', decimals: 18 },
  rpcUrls: {
    default: { http: [process.env.ORBIT_CHAIN_RPC] },
    public: { http: [process.env.ORBIT_CHAIN_RPC] },
  },
});

const orbitChainPublicClient = createPublicClient({ 
  chain: orbitChain, 
  transport: http() 
}).extend(arbOwnerPublicActions);

async function main() {
  const rollupCoreContracts = await createRollupFetchCoreContracts({
    rollup: process.env.ROLLUP_ADDRESS as `0x${string}`,
    publicClient: parentChainPublicClient,
  });

  const inbox = await parentChainPublicClient.readContract({
    address: rollupCoreContracts.rollup,
    abi: parseAbi(['function inbox() view returns (address)']),
    functionName: 'inbox',
  });

  const tokenBridgeContracts = await createTokenBridgeFetchTokenBridgeContracts({
    inbox,
    parentChainPublicClient,
  });

  // Step 1: Set L1 Price Per Unit to 0
  console.log('Setting L1PricePerUnit to 0...');
  const setL1PricePerUnitRequest =
    await orbitChainPublicClient.arbOwnerPrepareTransactionRequest({
      functionName: 'setL1PricePerUnit',
      args: [BigInt(0)],
      upgradeExecutor: tokenBridgeContracts.orbitChainContracts.upgradeExecutor,
      account: chainOwner.address,
    });

  await orbitChainPublicClient.sendRawTransaction({
    serializedTransaction: await chainOwner.signTransaction(setL1PricePerUnitRequest),
  });
  console.log('L1PricePerUnit set to 0');

  // Step 2: Set L1 Pricing Reward Rate to 0
  console.log('Setting L1PricingRewardRate to 0...');
  const setL1PricingRewardRateRequest =
    await orbitChainPublicClient.arbOwnerPrepareTransactionRequest({
      functionName: 'setL1PricingRewardRate',
      args: [BigInt(0)],
      upgradeExecutor: tokenBridgeContracts.orbitChainContracts.upgradeExecutor,
      account: chainOwner.address,
    });

  await orbitChainPublicClient.sendRawTransaction({
    serializedTransaction: await chainOwner.signTransaction(setL1PricingRewardRateRequest),
  });
  console.log('L1PricingRewardRate set to 0');
}

main();
```

### Why set both to 0?

When using **custom gas token** on AnyTrust chain:

- **SetL1PricePerUnit(0):** Disables L1 data cost component in gas pricing. Since you use custom gas token, you don't want ETH-based L1 pricing.

- **SetL1PricingRewardRate(0):** Disables L1 pricing rewards. The reward rate determines how much the network compensates batch posters for L1 costs. Set to 0 because your custom token chain handles this differently.

---

## Question 6: validation-server-configs-list setting

### JSON config example:

```json
{
  "node": {
    "block-validator": {
      "enable": true,
      "validation-server-configs-list": "[{\"jwtsecret\":\"/path/to/jwt.hex\",\"url\":\"ws://127.0.0.1:52000\"},{\"jwtsecret\":\"/path/to/jwt.hex\",\"url\":\"ws://127.0.0.1:52001\"}]"
    }
  }
}
```

### Command line example:

```bash
./nitro \
  --node.block-validator.enable=true \
  --node.block-validator.validation-server-configs-list='[{"jwtsecret":"/path/to/jwt.hex","url":"ws://127.0.0.1:52000"}]'
```

### Full setup with separate validation server:

**Step 1: Create JWT secret file**

```bash
openssl rand -hex 32 > /tmp/nitro-val.jwt
```

**Step 2: Start validation server (nitro-val)**

```bash
./nitro-val \
  --auth.addr=127.0.0.1 \
  --auth.port=52000 \
  --auth.jwtsecret=/tmp/nitro-val.jwt \
  --auth.origins=127.0.0.1 \
  --validation.wasm.allowed-wasm-module-roots=/path/to/machines
```

**Step 3: Start main node with validation server config**

```bash
./nitro \
  --node.block-validator.enable=true \
  --node.block-validator.validation-server-configs-list='[{"jwtsecret":"/tmp/nitro-val.jwt","url":"ws://127.0.0.1:52000"}]' \
  --validation.wasm.allowed-wasm-module-roots=/path/to/machines
```

### Multiple validation servers:

```json
{
  "node": {
    "block-validator": {
      "enable": true,
      "validation-server-configs-list": "[{\"jwtsecret\":\"/tmp/jwt1.hex\",\"url\":\"ws://validator1:52000\"},{\"jwtsecret\":\"/tmp/jwt2.hex\",\"url\":\"ws://validator2:52000\"}]"
    }
  }
}
```

**Code location:** `staker/block_validator.go:128`

---

## Question 7: v3.7.4 vs v3.7.5

### Differences between v3.7.4 and v3.7.5:

Based on git log, v3.7.5 includes these changes over v3.7.4:

1. **Beacon blob API support** - Add support for new beacon chain `/blobs` endpoint
2. **Remove pre-Stylus validation** - Cleanup of old validation code
3. **Better help text** - More descriptive help for beacon-url option

### Which version to use?

**Recommendation:** Use **v3.7.5** if possible.

- v3.7.5 has better beacon chain support
- v3.7.5 removed some deprecated code
- Both versions support BOLD

### BOLD config for v3.7.5:

Yes, you can use this config:

```json
{
  "node": {
    "bold": {
      "enable": true,
      "strategy": "MakeNodes"
    }
  }
}
```

**Note:** Make sure:
- "strategy" value should be "MakeNodes" or "Watchtower" (case sensitive)
- You need quotes around the strategy value
- The "enable" field might be needed depending on your setup

---

## Question 8: Decode batch poster transaction data

### Step-by-step guide to retrieve data from batch poster transaction:

**Step 1: Get the batch poster transaction**

```javascript
const txHash = '0x...'; // Your batch poster transaction hash
const tx = await provider.getTransaction(txHash);
const txData = tx.data;
```

**Step 2: Decode the function call**

The function signature is:
```
addSequencerL2BatchFromOrigin(uint256 sequenceNumber, bytes data, uint256 afterDelayedMessagesRead, address gasRefunder, uint256 prevMessageCount, uint256 newMessageCount)
```

Function selector: `0x8f111f3c`

```javascript
const ethers = require('ethers');

const seqInboxABI = [
  'function addSequencerL2BatchFromOrigin(uint256 sequenceNumber, bytes data, uint256 afterDelayedMessagesRead, address gasRefunder, uint256 prevMessageCount, uint256 newMessageCount)'
];

const iface = new ethers.utils.Interface(seqInboxABI);
const decoded = iface.decodeFunctionData('addSequencerL2BatchFromOrigin', txData);

const batchData = decoded.data; // This is the bytes data field
```

**Step 3: Parse the batch data**

```javascript
// batchData format:
// - First byte (1 byte): header flag
// - If header has DAS flag (0x80): next 32 bytes is keyset hash, then 32 bytes is data hash

const headerFlag = batchData[0];

// Check if it's a DAS batch (has bit 0x80)
const isDASBatch = (headerFlag & 0x80) !== 0;

if (isDASBatch) {
  console.log('This is a DAS batch');
  
  // Skip header (1 byte)
  // Next 32 bytes: keyset hash
  const keysetHash = '0x' + Buffer.from(batchData.slice(1, 33)).toString('hex');
  console.log('Keyset Hash:', keysetHash);
  
  // Next 32 bytes: data hash
  const dataHash = '0x' + Buffer.from(batchData.slice(33, 65)).toString('hex');
  console.log('Data Hash:', dataHash);
  
  // Use dataHash to retrieve data from DAS
  // curl http://YOUR_DAS_SERVER:9877/get-by-hash/0x{dataHash}
}
```

**Step 4: Retrieve data from DAS**

```bash
# Using the data hash from step 3
curl "http://YOUR_DAS_SERVER:9877/get-by-hash/0x{DATA_HASH_HERE}"
```

### Complete example:

```javascript
const ethers = require('ethers');

async function decodeBatchPosterTx(provider, txHash) {
  // Get transaction
  const tx = await provider.getTransaction(txHash);
  
  // Decode function call
  const seqInboxABI = [
    'function addSequencerL2BatchFromOrigin(uint256 sequenceNumber, bytes data, uint256 afterDelayedMessagesRead, address gasRefunder, uint256 prevMessageCount, uint256 newMessageCount)'
  ];
  const iface = new ethers.utils.Interface(seqInboxABI);
  
  try {
    const decoded = iface.decodeFunctionData('addSequencerL2BatchFromOrigin', tx.data);
    
    const batchData = ethers.utils.arrayify(decoded.data);
    const headerFlag = batchData[0];
    
    console.log('Sequence Number:', decoded.sequenceNumber.toString());
    console.log('Header Flag:', '0x' + headerFlag.toString(16));
    
    // Check header flag type
    // 0x80 = DAS without tree
    // 0x88 = DAS with tree (0x80 | 0x08)
    // 0x50 = Blob
    
    if ((headerFlag & 0x80) !== 0) {
      console.log('Type: DAS batch');
      
      const keysetHash = ethers.utils.hexlify(batchData.slice(1, 33));
      const dataHash = ethers.utils.hexlify(batchData.slice(33, 65));
      
      console.log('Keyset Hash:', keysetHash);
      console.log('Data Hash:', dataHash);
      
      return { type: 'DAS', keysetHash, dataHash };
    }
    
    return { type: 'Other', headerFlag };
    
  } catch (e) {
    // Try delay proof version
    const delayProofABI = [
      'function addSequencerL2BatchFromOriginDelayProof(uint256 sequenceNumber, bytes data, uint256 afterDelayedMessagesRead, address gasRefunder, uint256 prevMessageCount, uint256 newMessageCount, tuple(bytes32, tuple(uint8, address, uint64, uint64, uint256, uint256, bytes32)))'
    ];
    const iface2 = new ethers.utils.Interface(delayProofABI);
    const decoded = iface2.decodeFunctionData('addSequencerL2BatchFromOriginDelayProof', tx.data);
    
    // Same parsing logic...
  }
}
```

### Header flag values:

| Flag | Hex | Meaning |
|------|-----|---------|
| DAS (no tree) | 0x80 | DAS certificate |
| DAS (with tree) | 0x88 | DAS with tree merkelization |
| Blob | 0x50 | EIP-4844 blob data |
| Brotli | 0x00 | Brotli compressed |
| Zeroheavy | 0x20 | Zeroheavy encoded |

**Code location:** `daprovider/util.go:58-80`

---

## Question 9: @arbitrum/chain-sdk version

**Note:** The correct package name is `@arbitrum/orbit-sdk`, not `@arbitrum/chain-sdk`.

### Default version when running yarn add:

```bash
yarn add @arbitrum/orbit-sdk
```

This will download the latest version from npm. Currently the SDK version is **0.24.0**.

### Supported Nitro versions:

Based on SDK source code:

- **Default Nitro node image:** `offchainlabs/nitro-node:v3.6.0-fc07dd2`
- **Tested with Nitro contracts:** v3.1.1, v2.1.3
- **Supported ArbOS versions (wasm module roots):** 10, 10.1, 10.2, 10.3, 11, 11.1, 20, 30, 31, 32, 40

---

## Question 10: Relationship between ArbOS, Orbit, and Nitro versions

### Version relationships:

```
Nitro Version (e.g., v3.7.4)
    └── Contains ArbOS versions support (e.g., up to ArbOS 51)
    └── Used by Orbit chains

Orbit SDK Version (e.g., 0.24.0)
    └── Provides tools to deploy/manage Orbit chains
    └── Supports specific Nitro versions
```

### ArbOS vs Nitro:

| Aspect | ArbOS | Nitro |
|--------|-------|-------|
| What is it | L2 hypervisor/runtime | Full node software |
| Scope | Execution environment | Complete node (consensus + execution) |
| Updates | On-chain upgrades | Node software updates |
| Version format | Integer (20, 30, 40...) | Semver (v3.7.4, v3.8.0...) |

### Role differences:

**Nitro:**
- The complete node implementation
- Handles networking, consensus, state sync
- Can be upgraded by replacing binary

**ArbOS:**
- The on-chain execution layer
- Handles gas pricing, L1/L2 communication, block production
- Upgraded via on-chain governance

### Compatibility:

- Nitro node must support the chain's ArbOS version
- If chain upgrades to ArbOS 51, node must be at least v3.7.x
- Node version check: `MaxArbosVersionSupported` in `go-ethereum/params/config_arbitrum.go`

---

## Question 11: Orbit SDK default branch

### Is main the default branch?

**Yes**, the default branch for orbit-sdk is `main`.

Git config shows:
```
remotes/origin/HEAD -> origin/main
```

### Nitro version supported by default clone:

When you `git clone https://github.com/OffchainLabs/arbitrum-orbit-sdk.git`:

- You get the `main` branch
- SDK version: 0.24.0
- Default Nitro node version: `v3.6.0-fc07dd2`
- Supports Nitro contracts: v3.1.1, v2.1.3

### Check supported versions in code:

```bash
# Check wasm module roots (ArbOS versions)
cat src/wasmModuleRoot.ts

# Check node config defaults
cat src/types/NodeConfig.generated.ts
```

---

## Summary

| Question | Quick Answer |
|----------|--------------|
| 1-1 | Use `eth_syncing` RPC, returns `false` if synced |
| 1-2 | Use `--log-level=TRACE` |
| 1-3 | Compare block height/hash with other nodes |
| 2 | reorg-resistance-margin=0 disables L1 reorg protection |
| 3 | ArbOS 40 adds EIP-2935, MaxWasmSize config |
| 4 | Split into sequencer node + validator node with separate configs |
| 5 | Change functionName and args for SetL1PricePerUnit/SetL1PricingRewardRate |
| 6 | Use JSON array string for validation-server-configs-list |
| 7 | Use v3.7.5, it has better beacon chain support |
| 8 | Decode tx data, check header flag 0x80, extract keyset/data hash |
| 9 | Package is @arbitrum/orbit-sdk, current version 0.24.0 |
| 10 | Nitro is node software, ArbOS is on-chain runtime |
| 11 | Yes, main branch is default, supports Nitro v3.6.0 |
