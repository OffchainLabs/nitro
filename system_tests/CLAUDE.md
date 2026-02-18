# System Tests

Package `arbtest`. Integration tests that spin up actual node instances. Heavyweight compared to unit tests -- prefer unit tests for logic that doesn't require a running node.

Prerequisites: `make test-go-deps` must have run (builds WASM artifacts, stylus test wasms, replay environment).

## NodeBuilder

All tests that need a running node use `NodeBuilder` (defined in `common_test.go`).

### Basic setup

```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

builder := NewNodeBuilder(ctx).DefaultConfig(t, true)  // true = with L1
cleanup := builder.Build(t)
defer cleanup()
```

`DefaultConfig(t, withL1)`:
- `withL1 = true` -- deploys L1 + L2 with sequencer, staker, batch poster
- `withL1 = false` -- L2-only, no L1 interaction

`Build` calls `t.Parallel()` by default. Use `.DontParalellise()` to disable.

### Configuration

Chainable methods before `Build`:

```go
builder.WithArbOSVersion(params.ArbosVersion_30)
builder.WithPreBoldDeployment()       // legacy rollup (no BOLD)
builder.WithReferenceDA()             // enable ReferenceDA provider
builder.WithDelayBuffer(threshold)
builder.RequireScheme(t, rawdb.HashScheme)  // skip test if wrong scheme
```

Direct field assignment is also common:

```go
builder.nodeConfig.BlockValidator.Enable = true
builder.nodeConfig.BatchPoster.Enable = true
builder.execConfig.Sequencer.MaxRevertGasReject = 0
```

### Second node / L3

```go
nodeB, cleanupB := builder.Build2ndNode(t, &SecondNodeParams{
    nodeConfig: arbnode.ConfigDefaultL1NonSequencerTest(),
})
defer cleanupB()

// L3:
cleanupL3 := builder.BuildL3OnL2(t)
defer cleanupL3()
// Access via builder.L3, builder.L3Info
```

## Accounts

`BlockchainTestInfo` manages test accounts. Pre-created accounts:
- L2: `"Owner"`, `"Faucet"` (genesis accounts with large balances)
- L1: `"Faucet"`, `"RollupOwner"`, `"Sequencer"`, `"Validator"`, `"User"`

```go
builder.L2Info.GenerateAccount("Alice")
auth := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
addr := builder.L2Info.GetAddress("Alice")
tx := builder.L2Info.PrepareTx("Owner", "Alice", builder.L2Info.TransferGas, big.NewInt(1e12), nil)
```

## Common helpers

```go
Require(t, err)                                    // fail if err != nil
Fatal(t, args...)                                  // always fail

receipt, err := EnsureTxSucceeded(ctx, client, tx) // wait for tx, assert success
receipt := EnsureTxFailed(t, ctx, client, tx)      // wait for tx, assert failure
_, err = WaitForTx(ctx, client, txHash, timeout)   // wait for mining

builder.L2.DeploySimple(t, auth)                   // deploy test contract
GetBalance(t, ctx, client, addr)                   // check balance
AdvanceL1(t, ctx, l1Client, l1Info, numBlocks)     // mine L1 blocks
```

### Cross-chain helpers

```go
builder.BridgeBalance(t, "Faucet", big.NewInt(1e18))       // L1 -> L2 deposit
SendSignedTxViaL1(t, ctx, l1info, l1client, l2client, tx)  // delayed inbox
```

## Validation test setup

Tests involving block validation need extra setup:

```go
builder.RequireScheme(t, rawdb.HashScheme)  // validation requires hash scheme
valConf := valnode.TestValidationConfig
valConf.UseJit = true
_, valStack := createTestValidationNode(t, ctx, &valConf)
configByValidationNode(builder.nodeConfig, valStack)
```

## Live config updates

```go
updatedConfig := *builder.nodeConfig
updatedConfig.SomeField = newValue
builder.L2.ConsensusConfigFetcher.Set(&updatedConfig)
```
