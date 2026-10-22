// SPDX-License-Identifier: MIT
// solc not assumed available; the runtime bytecode is assembled at test time
// in `crates/arb-fuzz/src/arbitrary_impls/interop.rs::stylus_callback_runtime()`.
// This file is reference source for review.
pragma solidity ^0.8.20;

contract StylusCallback {
    uint256 public pingCount; // slot 0

    /// Increments pingCount and returns input + 1.
    function ping(uint256 input) external returns (uint256) {
        pingCount += 1;
        return input + 1;
    }
}
