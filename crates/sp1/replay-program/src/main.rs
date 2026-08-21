// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

#![cfg_attr(target_os = "zkvm", no_main)]

#[cfg(target_os = "zkvm")]
sp1_zkvm::entrypoint!(main);

use replay_io::recv::ValidationPayload;

fn main() {
    let _inputs = replay_io::recv::bootload_inputs();

    sp1_zkvm::syscalls::syscall_dump_elf();

    match replay_io::recv::validation_payload() {
        ValidationPayload::Bootload => sp1_zkvm::syscalls::syscall_halt(0),
        ValidationPayload::Input(_) => {
            println!("Validation MOCKED with hash 0x{}", "00".repeat(32));
        }
    }
}
