// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/bold/blob/main/LICENSE
// SPDX-License-Identifier: BUSL-1.1
//
pragma solidity ^0.8.17;

import "forge-std/Test.sol";
import "../../src/state/GlobalState.sol";
import "../../src/state/Machine.sol";
import "../../src/rollup/RollupLib.sol";
import "./Utils.sol";

library StateToolsLib {
    using GlobalStateLib for GlobalState;

    function randomState(Random rand, uint256 inboxMsgCountProcessed, bytes32 blockHash, MachineStatus ms)
        internal
        returns (AssertionState memory)
    {
        bytes32[2] memory bytes32Vals = [blockHash, rand.hash()];
        uint64[2] memory u64Vals = [uint64(inboxMsgCountProcessed), uint64(uint256(rand.hash()))];

        GlobalState memory gs = GlobalState({bytes32Vals: bytes32Vals, u64Vals: u64Vals});
        return AssertionState({globalState: gs, machineStatus: ms, endHistoryRoot: bytes32(0)});
    }

    function hash(AssertionState memory s) internal pure returns (bytes32) {
        return s.globalState.hash();
    }

    function mockMachineHash(AssertionState memory s) internal pure returns (bytes32) {
        return s.globalState.hash();
    }
}
