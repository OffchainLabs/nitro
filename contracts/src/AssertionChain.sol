// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.13;

interface IAssertionChain {
    struct Assertion {
        uint256 seqNum;
        StateCommitment stateCommitment;
        uint256 status;
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

    // Read-only calls.
    function numAssertions() external view returns (uint256);
    function challengePeriodSeconds() external view returns (uint256);
    function latestConfirmedAssertion() external view returns (Assertion memory assertion);
    function getAssertion(uint256 seqNum) external view returns (Assertion memory assertion);
    function getChallenge(bytes32 parentStateCommitHash) external view returns (Challenge memory challenge);
    function getChallengeVertex(uint256 seqNum, bytes32 parentStateCommitHash)
        external
        view
        returns (ChallengeVertex memory vertex);
    function challengeWinner(Challenge memory challenge) external returns (Assertion memory assertion);
    function challengeCompleted(Challenge memory challenge) external returns (bool);
    function eligibleForNewSuccessor(ChallengeVertex memory vertex) external returns (bool);
    function isPresumptiveSuccessor(ChallengeVertex memory vertex) external returns (bool);

    // Mutating calls.
    function createAssertion(Assertion calldata prev, StateCommitment calldata commit)
        external
        payable
        returns (Assertion memory assertion);
    function confirmForWin(Assertion calldata assertion) external payable;
    function confirmNoRival(Assertion calldata assertion) external payable;
    function rejectForLoss(Assertion calldata assertion) external payable;
    function rejectForPrev(Assertion calldata assertion) external payable;
    function confirmForPSTimer(ChallengeVertex calldata vertex) external payable;
    function confirmForChallengeDeadline(ChallengeVertex calldata vertex) external payable;
    function confirmForSubchallengeWin(ChallengeVertex calldata vertex) external payable;

    function createChallenge(Assertion calldata prev) external payable returns (Challenge memory challenge);
    function addChallengeVertex(Assertion calldata assertion, HistoryCommitment calldata history)
        external
        payable
        returns (ChallengeVertex memory vertex);
    function bisect(ChallengeVertex calldata vertex, HistoryCommitment calldata history, bytes32[] calldata proof)
        external
        payable
        returns (ChallengeVertex memory bisectedVertex);
    function merge(ChallengeVertex calldata mergingFrom, ChallengeVertex calldata mergingTo, bytes32[] calldata proof)
        external
        payable
        returns (ChallengeVertex memory mergedToVertex);
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
        revert("unimplemented");
    }

    function getAssertion(uint256 seqNum) external view returns (Assertion memory assertion) {
        revert("unimplemented");
    }

    function getChallenge(bytes32 parentStateCommitHash) external view returns (Challenge memory challenge) {
        revert("unimplemented");
    }

    function getChallengeVertex(uint256 seqNum, bytes32 parentStateCommitHash)
        external
        view
        returns (ChallengeVertex memory vertex)
    {
        revert("unimplemented");
    }

    function challengeWinner(Challenge memory challenge) external returns (Assertion memory assertion) {
        revert("unimplemented");
    }

    function challengeCompleted(Challenge memory challenge) external returns (bool) {
        return false;
    }

    function eligibleForNewSuccessor(ChallengeVertex memory vertex) external returns (bool) {
        return false;
    }

    function isPresumptiveSuccessor(ChallengeVertex memory vertex) external returns (bool) {
        return false;
    }

    // Mutating calls.
    function createAssertion(Assertion calldata prev, StateCommitment calldata commit)
        external
        payable
        returns (Assertion memory assertion)
    {
        revert("unimplemented");
    }

    function confirmForWin(Assertion calldata assertion) external payable {
        revert("unimplemented");
    }

    function confirmNoRival(Assertion calldata assertion) external payable {
        revert("unimplemented");
    }

    function rejectForLoss(Assertion calldata assertion) external payable {
        revert("unimplemented");
    }

    function rejectForPrev(Assertion calldata assertion) external payable {
        revert("unimplemented");
    }

    function confirmForPSTimer(ChallengeVertex calldata vertex) external payable {
        revert("unimplemented");
    }

    function confirmForChallengeDeadline(ChallengeVertex calldata vertex) external payable {
        revert("unimplemented");
    }

    function confirmForSubchallengeWin(ChallengeVertex calldata vertex) external payable {
        revert("unimplemented");
    }

    function createChallenge(Assertion calldata prev) external payable returns (Challenge memory challenge) {
        revert("unimplemented");
    }

    function addChallengeVertex(Assertion calldata assertion, HistoryCommitment calldata history)
        external
        payable
        returns (ChallengeVertex memory vertex)
    {
        revert("unimplemented");
    }

    function bisect(ChallengeVertex calldata vertex, HistoryCommitment calldata history, bytes32[] calldata proof)
        external
        payable
        returns (ChallengeVertex memory bisectedVertex)
    {
        revert("unimplemented");
    }

    function merge(ChallengeVertex calldata mergingFrom, ChallengeVertex calldata mergingTo, bytes32[] calldata proof)
        external
        payable
        returns (ChallengeVertex memory mergedToVertex)
    {
        revert("unimplemented");
    }
}
