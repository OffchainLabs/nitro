// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

use sp1_build::build_program_with_args;

fn main() {
    build_program_with_args("../stylus-compiler-program", Default::default())
}
