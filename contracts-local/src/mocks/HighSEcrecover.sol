// Copyright 2026, Offchain Labs, Inc.
// For license information, see:
// https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.0;

// Test contract for the ECRECOVER high-S divergence bug between native Go execution
// (accepts high-S) and the WASM prover / k256 (rejects high-S).
contract HighSEcrecover {
    // 1 = ECRECOVER accepted (non-zero address returned)
    // 2 = ECRECOVER rejected (address(0) returned)
    uint256 public result;

    // Accepts 128 bytes in ECRECOVER precompile format: hash | v | r | s (each 32 bytes).
    fallback() external {
        bytes32 hash = bytes32(msg.data[0:32]);
        uint8 v = uint8(uint256(bytes32(msg.data[32:64])));
        bytes32 r = bytes32(msg.data[64:96]);
        bytes32 s = bytes32(msg.data[96:128]);
        address addr = ecrecover(hash, v, r, s);
        result = addr != address(0) ? 1 : 2;
    }
}
