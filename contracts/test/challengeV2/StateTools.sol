// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.17;

import "forge-std/Test.sol";
import "../../src/state/GlobalState.sol";
import "../../src/state/Machine.sol";
import "./Utils.sol";

struct State {
    GlobalState gs;
    uint256 inboxMsgCountMax;
    MachineStatus ms;
}

library StateToolsLib {
    using GlobalStateLib for GlobalState;

    function randomState(Random rand, uint256 inboxMsgCountProcessed, bytes32 blockHash, MachineStatus ms)
        internal
        returns (State memory)
    {
        bytes32[2] memory bytes32Vals = [blockHash, rand.hash()];
        uint64[2] memory u64Vals = [uint64(inboxMsgCountProcessed), uint64(uint256(rand.hash()))];

        GlobalState memory gs = GlobalState({bytes32Vals: bytes32Vals, u64Vals: u64Vals});

        return State({gs: gs, inboxMsgCountMax: inboxMsgCountProcessed + 3, ms: ms});
    }

    function hash(State memory s) internal pure returns (bytes32) {
        // CHRIS: TODO: for some reason importing the RollupLib causes compilation failure - perhaps circular
        // CHRIS: TODO: we should transition to the rollup lib when this is fixed though
        return keccak256(abi.encodePacked(s.gs.hash(), s.inboxMsgCountMax, s.ms));
    }
}
