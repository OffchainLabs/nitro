// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.17;

library HistoryCommitmentLib {
    function hasState(bytes32 historyCommitment, bytes32 state, uint256 stateHeight, bytes memory proof)
        internal
        pure
        returns (bool)
    {
        // CHRIS: TODO: do a merkle proof check
        return true;
    }

    function hasPrefix(
        bytes32 historyCommitment,
        bytes32 prefixHistoryCommitment,
        uint256 prefixHistoryHeight,
        bytes memory proof
    ) internal pure returns (bool) {
        // CHRIS: TODO:
        // prove that the sequence of states commited to by prefixHistoryCommitment is a prefix
        // of the sequence of state commited to by the historyCommitment
        return true;
    }
}
