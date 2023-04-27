// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.17;

import "forge-std/Test.sol";
import "../../src/state/GlobalState.sol";
import "../../src/state/Machine.sol";
import "../../src/rollup/RollupLib.sol";
import "./Utils.sol";

struct State {
    ExecutionState es;
    uint256 inboxMsgCountMax;
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
        ExecutionState memory es = ExecutionState({globalState: gs, machineStatus: ms});

        return State({es: es, inboxMsgCountMax: inboxMsgCountProcessed + 3});
    }

    function hash(State memory s) internal pure returns (bytes32) {
        return RollupLib.stateHashMem(s.es, s.inboxMsgCountMax);
    }

    function mockMachineHash(State memory s) internal pure returns (bytes32) {
        return s.es.globalState.hash();
    }
}
