// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

use sp1_build::{BuildArgs, build_program_with_args};

fn main() {
    for program in ["../stylus-compiler-program", "../replay-program"] {
        build_program_with_args(
            program,
            BuildArgs {
                locked: true,
                ..Default::default()
            },
        )
    }
}
