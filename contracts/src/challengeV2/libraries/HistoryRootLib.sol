// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.17;

library HistoryRootLib {
    function hasState(bytes32 historyRoot, bytes32 state, uint256 stateHeight, bytes memory proof)
        internal
        pure
        returns (bool)
    {
        // CHRIS: TODO: do a merkle proof check
        return true;
    }

    function hasPrefix(
        bytes32 historyRoot,
        bytes32 prefixHistoryRoot,
        uint256 prefixHistoryHeight,
        bytes memory proof
    ) internal pure returns (bool) {
        // CHRIS: TODO:
        // prove that the sequence of states commited to by prefixHistoryRoot is a prefix
        // of the sequence of state commited to by the historyRoot
        return true;
    }
}
