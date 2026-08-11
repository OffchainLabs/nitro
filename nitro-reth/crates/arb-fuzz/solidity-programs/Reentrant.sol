// SPDX-License-Identifier: MIT
// solc not assumed available; the runtime bytecode is assembled at test time
// in `crates/arb-fuzz/src/arbitrary_impls/interop.rs::reentrant_runtime()`.
// This file is reference source for review.
pragma solidity ^0.8.20;

interface ISolCaller {
    function forward(address target, bytes calldata data) external returns (bytes memory);
}

contract Reentrant {
    /// Calls back into the Stylus caller's `forward(address,bytes)` selector,
    /// triggering a cross-language re-entry.
    function attack(address stylus) external returns (uint256) {
        // Encode forward(this, 0x) — self-target with empty data.
        bytes memory data = abi.encodeWithSelector(
            ISolCaller.forward.selector,
            address(this),
            ""
        );
        (bool ok, ) = stylus.call(data);
        require(ok, "reentry failed");
        return 1;
    }
}
