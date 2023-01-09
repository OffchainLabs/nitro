// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.13;

interface IAssertionChain {
    struct Assertion {
        uint256 seqNum;
        StateCommitment stateCommitment;
        uint status;
        bool isFirstChild;
        uint256 firstChildCreationTimestamp;
        uint256 secondChildCreationTimestamp;
        address actor;
    }
    struct Challenge {
        uint256 seqNum;
        uint256 nextSeqNum;
        ChallengeVertex root;
        ChallengeVertex latestConfirmed;
        uint256 creationTimestamp;
        address actor;
    }
    struct ChallengeVertex {
        uint256 seqNum;
        bytes32 challengeParentStateCommitHash;
        bool actor;
        bool isLeaf;
        uint256 psTimer;
    }
    struct StateCommitment {
        uint256 height;
        bytes32 stateRoot;
    }
    struct HistoryCommitment {
        uint256 height;
        bytes32 merkleRoot;
    }
    function numAssertions() external view returns (uint256);
    function challengePeriodSeconds() external view returns (uint256);
    function latestConfirmedAssertion() external view returns (Assertion memory assertion);
    function getAssertion(uint256 seqNum) external view returns (Assertion memory assertion);
    function getChallenge(bytes32 parentStateCommitHash) external view returns (Challenge memory challenge);
    function getChallengeVertex(uint256 seqNum, bytes32 parentStateCommitHash) external view returns (ChallengeVertex memory vertex);

    // Creates an assertion in the protocol by committing to state root.
    function createAssertion(
        Assertion calldata prev,
        StateCommitment calldata commit
    ) external payable returns (Assertion memory assertion);

    function confirmForWin(Assertion calldata assertion) external payable;
    function confirmNoRival(Assertion calldata assertion) external payable;
    function rejectForLoss(Assertion calldata assertion) external payable;
    function rejectForPrev(Assertion calldata assertion) external payable;
    function confirmForPSTimer(ChallengeVertex calldata vertex) external payable;
    function confirmForChallengeDeadline(ChallengeVertex calldata vertex) external payable;
    function confirmForSubchallengeWin(ChallengeVertex calldata vertex) external payable;

    // Initiates a challenge on an assertion in the protocol.
    function createChallenge(Assertion calldata prev) external payable returns (Challenge memory challenge);
}

contract AssertionChain is IAssertionChain {
    // Read-only calls.
    function numAssertions() external view returns (uint256) {
        return 0;
    }
    function challengePeriodSeconds() external view returns (uint256) {
        return 0;
    }
    function latestConfirmedAssertion() external view returns (Assertion memory assertion) {
        revert("failed");
    }
    function getAssertion(uint256 seqNum) external view returns (Assertion memory assertion) {
        revert("failed");
    }
    function getChallenge(bytes32 parentStateCommitHash) external view returns (Challenge memory challenge) {
        revert("failed");
    }
    function getChallengeVertex(uint256 seqNum, bytes32 parentStateCommitHash) external view returns (ChallengeVertex memory vertex) {
        revert("failed");
    }

    // Mutating calls.
    function createAssertion(
        Assertion calldata prev,
        StateCommitment calldata commit
    ) external payable returns (Assertion memory assertion) {
        revert("failed");
    }
    function confirmForWin(Assertion calldata assertion) external payable {
        revert("failed");
    }
    function confirmNoRival(Assertion calldata assertion) external payable {
        revert("failed");
    }
    function rejectForLoss(Assertion calldata assertion) external payable {
        revert("failed");
    }
    function rejectForPrev(Assertion calldata assertion) external payable {
        revert("failed");
    }
    function confirmForPSTimer(ChallengeVertex calldata vertex) external payable {
        revert("failed");
    }
    function confirmForChallengeDeadline(ChallengeVertex calldata vertex) external payable {
        revert("failed");
    }
    function confirmForSubchallengeWin(ChallengeVertex calldata vertex) external payable {
        revert("failed");
    }
    function createChallenge(Assertion calldata prev) external payable returns (Challenge memory challenge) {
        revert("failed");
    }
}
