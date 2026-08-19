// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

#![cfg_attr(target_os = "zkvm", no_main)]

#[cfg(target_os = "zkvm")]
sp1_zkvm::entrypoint!(main);

use validation::SP1_BOOTLOAD_SENTINEL;

fn main() {
    // Input 1: replay.wasmu, read in place (~140MB, so no copying). Unused
    // until the guest hosts replay.wasm in wasmer.
    let sp1_zkvm::ReadVecResult { ptr, .. } = sp1_zkvm::read_vec_raw();
    assert!(!ptr.is_null());

    // Input 2: replay.wasm function names for profiler symbols. Unused until
    // then.
    let sp1_zkvm::ReadVecResult { ptr, .. } = sp1_zkvm::read_vec_raw();
    assert!(!ptr.is_null());

    // Bootloading dumps the initialized guest here; the dumped ELF resumes
    // below, reading its next input from the runner.
    sp1_zkvm::syscalls::syscall_dump_elf();

    // Input 3: the bootload sentinel (halt cleanly; the dump above is the
    // product) or, once ported, the rkyv ValidationInput.
    let input = sp1_zkvm::io::read::<Vec<u8>>();
    if input.as_slice() == SP1_BOOTLOAD_SENTINEL {
        sp1_zkvm::syscalls::syscall_halt(0);
    }

    println!("Validation MOCKED with hash 0x{}", "00".repeat(32));
}
