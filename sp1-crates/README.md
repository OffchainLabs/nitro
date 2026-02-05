This folder adds a new Arbitrum block runner(like `jit` and `prover` in `arbitrator` folder), powered by [Succinct SP1](https://github.com/succinctlabs/sp1). For now we are only showcasing the executor powered by SP1, but the same code should be able to generate zero knowledge proofs for validating an Arbitrum block.

Ideally, the crates in this folder shall be merged into `arbitrator`. But for now, the differences in wasmer and Rust versions prevent us from doing this. Later when all the issues have been tackled we will likely merge both Rust workspaces.

## Usage

To build this runner, of course you must be able to build a significant part of nitro. Please refer to [this document](https://docs.arbitrum.io/run-arbitrum-node/nitro/build-nitro-locally) on setting up the computer to build nitro.

One major dependency wasmer also requires you to have LLVM 21 installed. Please note that you must have LLVM 21.x.y installed. Wasmer requires this major version.

The code included here requires a SP1 Rust toolchain with riscv64 support. You can use the following commands:

```bash
$ curl -L https://sp1up.succinct.xyz | bash
$ sp1up -v v6.0.0-beta.1
```

To verify that correct toolchain have been installed, you can run the following 2 commands, and compare output:

```bash
$ rustc +succinct --version
rustc 1.93.0-dev
$ rustc +succinct --print target-list | grep succinct
riscv32im-succinct-zkvm-elf
riscv64im-succinct-zkvm-elf
```

Now you can start the build process:

```bash
$ git clone https://github.com/OffchainLabs/nitro
$ # Checkout current PR
$ ./sp1-crates/build.sh
```

To make things easier to understand, for now we are using a simple bash script. As the code grow more mature, we would merge the bash script into the top-level makefile.

After the build process, all the files required by SP1 runner can be found in `target/sp1` folder. Some important files are:

* `replay.wasm`: this is Arbitrum's STF built with `sp1` tag enabled, so it contains SP1 specific optimizations
* `dumped_replay_wasm.elf`: this is the SP1 version of `replay.wasm`. The 2 fulfill the same task. It's just that `dumped_replay_wasm.elf` is in RISC-V architecture accepted by SP1, following SP1's ABIs.
* `stylus-compiler-program`: this is a SP1 program wrapping wasmer's singlepass compiler. It can compile an Arbitrum stylus program into RISC-V format accepted by SP1.
* `sp1-runner`: this is the SP1 runner that can execute / validate an Arbitrum block. One can see it as SP1 version of `jit` or `prover` in `arbitrator` workspace.
* `*.json`: sample Arbitrum test blocks you can use for testing. In my run, `block_inputs_7.json` was generated.

There are also other files in the folder, but you can safely ignore them. They are kept now for debugging purposes.

Now you can use the following command to execute an Arbitrum block using SP1 runner:

```bash
$ ./target/sp1/sp1-runner \
    --program target/sp1/dumped_replay_wasm.elf \
    --stylus-compiler-program target/sp1/stylus-compiler-program \
    --block-file target/sp1/block_inputs_7.json
stderr: WARNING: Using insecure random number generator.
stdout: Validation succeeds with hash 624b2d504238ba9fe94ad3e19d1036a51894bc209b7f0ead1331d22005d40178
```

You can tweak `RUST_LOG` for more logs(e.g., running cycles and running time):

```bash
$ RUST_LOG=info ./target/sp1/sp1-runner \
    --program target/sp1/dumped_replay_wasm.elf \
    --stylus-compiler-program target/sp1/stylus-compiler-program \
    --block-file target/sp1/block_inputs_7.json
```

You can also compare the result hash with Arbitrum's own JIT engine:

```bash
$ ./target/bin/jit \
    --debug --cranelift \
    --binary target/machines/latest/replay.wasm \
    --json-inputs=target/sp1/block_inputs_7.json
Created the machine in 4081ms.
Completed in 19ms with hash 624b2d504238ba9fe94ad3e19d1036a51894bc209b7f0ead1331d22005d40178.
```

You can also generate other Arbitrum blocks to test. There is one caveat: for any stylus programs that might be executed, you must include the original wasm source in the block JSON file as well. SP1 runner works either with the original WASM source files(it will invoke stylus compiler program automatically), or the `rv64` target binary after compilation. You can also refer to [this commit](https://github.com/wakabat/nitro/commit/d924d4907ace4bd329b27dacd31ebad832b6eb90) where we patch nitro's tests to dump as many acceptable test blocks as possible.
