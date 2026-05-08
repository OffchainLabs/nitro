This folder adds a new Arbitrum block runner (like `jit` and `prover`), powered by [Succinct SP1](https://github.com/succinctlabs/sp1). For now we are only showcasing the executor powered by SP1, but the same code should be able to generate zero knowledge proofs for validating an Arbitrum block.

These crates are part of the root Cargo workspace.

## Usage

To build this runner, of course you must be able to build a significant part of nitro. Please refer to [this document](https://docs.arbitrum.io/run-arbitrum-node/nitro/build-nitro-locally) on setting up the computer to build nitro.

One major dependency wasmer also requires you to have LLVM 21 installed. Please note that you must have LLVM 21.x.y installed. Wasmer requires this major version.

The code included here requires a SP1 Rust toolchain with riscv64 support. You can install it via the Makefile target:

```bash
$ make -C crates/sp1 install-sp1
```

This runs `sp1up` and installs the RISC-V C toolchain under `$HOME/.sp1/`.

To verify that correct toolchain have been installed, you can run the following 2 commands, and compare output:

```bash
$ rustc +succinct --version
rustc 1.93.0-dev
$ rustc +succinct --print target-list | grep succinct
riscv32im-succinct-zkvm-elf
riscv64im-succinct-zkvm-elf
```

All SP1-related build steps are driven by `crates/sp1/Makefile`. Run `make -C crates/sp1 help` to see every available target. The main ones are:

* `install-sp1` — install the SP1 toolchain (see above).
* `brotli` — build brotli for the RISC-V target; artifacts go to `target/lib-sp1`.
* `nitro-deps` — build nitro dependencies needed by the SP1 runner (depends on `brotli`).
* `build` — build the SP1 runner and its inputs; artifacts go to `target/sp1`.
* `record-blocks` — run all block-recording system tests and dump block inputs to `target/sp1/block-inputs`.
* `run` — execute `sp1-runner` on a recorded block; defaults to `stylus`, override with `BLOCK=<name>`.
* `clean` — remove everything produced by the targets above.

Now you can start the build process:

```bash
$ git clone https://github.com/OffchainLabs/nitro
$ # Checkout current PR
$ make -C crates/sp1 build
```

After the build process, all the files required by SP1 runner can be found in `target/sp1` folder. Some important files are:

* `replay.wasm`: this is Arbitrum's STF, used as input to the SP1 builder.
* `dumped_replay_wasm.elf`: this is the SP1 version of `replay.wasm`. The 2 fulfill the same task. It's just that `dumped_replay_wasm.elf` is in RISC-V architecture accepted by SP1, following SP1's ABIs.
* `stylus-compiler-program`: this is a SP1 program wrapping wasmer's singlepass compiler. It can compile an Arbitrum stylus program into RISC-V format accepted by SP1.
* `sp1-runner`: this is the SP1 runner that can execute / validate an Arbitrum block. One can see it as SP1 version of `jit` or `prover` in `arbitrator` workspace.

There are also other files in the folder, but you can safely ignore them. They are kept now for debugging purposes.

To generate sample Arbitrum test blocks you can use for testing, run:

```bash
$ make -C crates/sp1 record-blocks
```

This runs the block-recording system tests (under the `block_recording` build tag) and writes the resulting JSON block inputs to `target/sp1/block-inputs/`:

* `transfer.json` — a pure ETH transfer, no contract interactions.
* `solidity.json` — 20 Solidity SSTORE operations, no Stylus.
* `stylus.json` — a single Stylus call with one WASM storage write.
* `stylus_heavy.json` — 32 cross-contract read/write pairs through a multicall Stylus program.
* `mixed.json` — a mixed block: ETH transfers, an EVM call, and multiple Stylus programs.
* `signatures.json` — 50 ETH transfers in a single block; amplifies ECRecover (sender recovery) signal in profile snapshots.

Now you can use the following command to execute an Arbitrum block using SP1 runner:

```bash
$ make -C crates/sp1 run BLOCK=stylus
stderr: WARNING: Using insecure random number generator.
stdout: Validation succeeds with hash 624b2d504238ba9fe94ad3e19d1036a51894bc209b7f0ead1331d22005d40178
```

Pass `BLOCK=<name>` to run against a different recorded block (for example `BLOCK=mixed`). You can also tweak `RUST_LOG` for more logs (e.g., running cycles and running time):

```bash
$ RUST_LOG=info make -C crates/sp1 run BLOCK=mixed
```

You can also compare the result hash with Arbitrum's own JIT engine:

```bash
$ ./target/bin/jit \
    --debug --cranelift \
    --binary target/machines/latest/replay.wasm \
    json --inputs=target/sp1/block-inputs/stylus.json
Created the machine in 4081ms.
Completed in 19ms with hash 624b2d504238ba9fe94ad3e19d1036a51894bc209b7f0ead1331d22005d40178.
```

If you want to generate additional Arbitrum blocks beyond the ones recorded by `make record-blocks`, you can add new tests to `system_tests/block_recording_test.go` (guarded by the `block_recording` build tag). There is one caveat: for any stylus programs that might be executed, you must include the original wasm source in the block JSON file as well. SP1 runner works either with the original WASM source files (it will invoke stylus compiler program automatically), or the `rv64` target binary after compilation.

## Profiling

Three Make targets drive `profile.py` and `profile_delta.py` in `crates/sp1/`:

* `profile` — single ad-hoc run. Builds a profiling-instrumented `sp1-runner`, executes every recorded block, and prints a human-readable summary table. Pass `JSON_OUTPUT=<path>` to also write a JSON snapshot.
* `profile-snapshot LABEL=<name>` — wrapper that runs `profile` and stores the JSON snapshot at `target/sp1/profile-<name>.json`.
* `profile-compare BASE=<old> NEW=<new>` — runs `profile_delta.py` against two saved snapshots and writes a markdown delta report to `target/sp1/profile-delta-<old>-vs-<new>.md` (also `cat`s it).

Each block's JSON snapshot includes per-phase `cycles`, `gas`, `wasm_size`, `time_secs`, plus the `syscalls` map (`SECP256K1_ADD`, `KECCAK_PERMUTE`, `MPROTECT`, …) and any `cycle_trackers` spans emitted by the runner. The delta report renders a per-block table for those scalars plus a separate syscall diff table (with crypto syscalls listed first) and a cycle-tracker diff table when present.

### Typical "before / after" workflow

When evaluating a change to the SP1 program (a library swap, new precompile, optimization), use snapshots to capture the impact:

```bash
# 1. Record block inputs once (only needs to be redone when block_recording_test.go changes).
$ make -C crates/sp1 record-blocks

# 2. On the baseline branch, capture a snapshot.
$ git checkout master
$ make -C crates/sp1 build
$ make -C crates/sp1 profile-snapshot LABEL=baseline

# 3. On the change branch, capture another snapshot.
$ git checkout my-change
$ make -C crates/sp1 build
$ make -C crates/sp1 profile-snapshot LABEL=my-change

# 4. Render the delta report.
$ make -C crates/sp1 profile-compare BASE=baseline NEW=my-change
```

Snapshots persist under `target/sp1/profile-*.json`, so it's safe to switch branches between steps 2 and 3 — the JSON files are not removed by `make build`. Only `make clean` wipes them.

### Sanity checks on a fresh snapshot

```bash
# Confirm the new fields are populated.
$ jq '.blocks.transfer.reexecution | keys' target/sp1/profile-baseline.json
[ "cycles", "cycle_trackers", "gas", "syscalls", "time_secs" ]

# Confirm SECP256K1_* syscalls appear (one per transaction in the block).
$ jq '.blocks.signatures.reexecution.syscalls | with_entries(select(.key | startswith("SECP256K1")))' \
    target/sp1/profile-baseline.json
```

The `signatures` block is the most ECRecover-heavy input (~50 × per-tx SECP syscalls); use it when measuring changes that touch `arbcrypto::ecrecovery` or the SP1 `secp256k1` / `k256` patches.
