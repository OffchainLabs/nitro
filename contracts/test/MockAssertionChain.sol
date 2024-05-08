// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.17;

import "forge-std/Test.sol";
import {IAssertionChain} from "../src/challengeV2/IAssertionChain.sol";
import { IEdgeChallengeManager } from "../src/challengeV2/EdgeChallengeManager.sol";
import "../src/bridge/IBridge.sol";
import "../src/rollup/RollupLib.sol";
import "./challengeV2/StateTools.sol";

struct MockAssertion {
    bytes32 predecessorId;
    uint256 height;
    AssertionState state;
    bytes32 successionChallenge;
    uint64 firstChildCreationBlock;
    uint64 secondChildCreationBlock;
    bool isFirstChild;
    bool isPending;
    bytes32 configHash;
}

contract MockAssertionChain is IAssertionChain {
    mapping(bytes32 => MockAssertion) assertions;
    IBridge public bridge; // TODO: set bridge in this mock
    bytes32 public wasmModuleRoot;
    uint256 public baseStake;
    address public challengeManager;
    uint64 public confirmPeriodBlocks;

    bool public validatorWhitelistDisabled;
    mapping(address => bool) public isValidator;

    function assertionExists(bytes32 assertionHash) public view returns (bool) {
        return assertions[assertionHash].height != 0;
    }

    function stakeToken() public view returns(address) {
        return address(0);
    }

    function validateAssertionHash(
        bytes32 assertionHash,
        AssertionState calldata state,
        bytes32 prevAssertionHash,
        bytes32 inboxAcc
    ) external view {
        require(assertionExists(assertionHash), "Assertion does not exist");
        // TODO: HN: This is not how the real assertion chain calculate assertion hash
        require(assertionHash == calculateAssertionHash(prevAssertionHash, state), "INVALID_ASSERTION_HASH");
    }

    function getFirstChildCreationBlock(bytes32 assertionHash) external view returns (uint64) {
        require(assertionExists(assertionHash), "Assertion does not exist");
        return assertions[assertionHash].firstChildCreationBlock;
    }

    function getSecondChildCreationBlock(bytes32 assertionHash) external view returns (uint64) {
        require(assertionExists(assertionHash), "Assertion does not exist");
        return assertions[assertionHash].secondChildCreationBlock;
    }

    function validateConfig(
        bytes32 assertionHash,
        ConfigData calldata configData
    ) external view {
        require(
            RollupLib.configHash({
                wasmModuleRoot: configData.wasmModuleRoot,
                requiredStake: configData.requiredStake,
                challengeManager: configData.challengeManager,
                confirmPeriodBlocks: configData.confirmPeriodBlocks,
                nextInboxPosition: configData.nextInboxPosition
            }) == assertions[assertionHash].configHash,
            "BAD_CONFIG"
        );
    }

    function isFirstChild(bytes32 assertionHash) external view returns (bool) {
        require(assertionExists(assertionHash), "Assertion does not exist");
        return assertions[assertionHash].isFirstChild;
    }

    function isPending(bytes32 assertionHash) external view returns (bool) {
        require(assertionExists(assertionHash), "Assertion does not exist");
        return assertions[assertionHash].isPending;
    }

    function calculateAssertionHash(
        bytes32 predecessorId,
        AssertionState memory afterState
    )
        public
        pure
        returns (bytes32)
    {
        return RollupLib.assertionHash({
            parentAssertionHash: predecessorId,
            afterState: afterState,
            inboxAcc: keccak256(abi.encode(afterState.globalState.u64Vals[0])) // mock accumulator based on inbox count
        });
    }

    function childCreated(bytes32 assertionHash) internal {
        if (assertions[assertionHash].firstChildCreationBlock == 0) {
            assertions[assertionHash].firstChildCreationBlock = uint64(block.number);
        } else if (assertions[assertionHash].secondChildCreationBlock == 0) {
            assertions[assertionHash].secondChildCreationBlock = uint64(block.number);
        }
    }

    function addAssertionUnsafe(
        bytes32 predecessorId,
        uint256 height,
        uint64 nextInboxPosition,
        AssertionState memory afterState,
        bytes32 successionChallenge
    ) public returns (bytes32) {
        bytes32 assertionHash = calculateAssertionHash(predecessorId, afterState);
        assertions[assertionHash] = MockAssertion({
            predecessorId: predecessorId,
            height: height,
            state: afterState,
            successionChallenge: successionChallenge,
            firstChildCreationBlock: 0,
            secondChildCreationBlock: 0,
            isFirstChild: assertions[predecessorId].firstChildCreationBlock == 0,
            isPending: true,
            configHash: RollupLib.configHash({
                wasmModuleRoot: wasmModuleRoot,
                requiredStake: baseStake,
                challengeManager: challengeManager,
                confirmPeriodBlocks: confirmPeriodBlocks,
                nextInboxPosition: nextInboxPosition
            })
        });
        childCreated(predecessorId);
        return assertionHash;
    }

    function addAssertion(
        bytes32 predecessorId,
        uint256 height,
        uint64 nextInboxPosition,
        AssertionState memory beforeState,
        AssertionState memory afterState,
        bytes32 successionChallenge
    ) public returns (bytes32) {
        bytes32 beforeStateHash = StateToolsLib.hash(beforeState);
        bytes32 assertionHash = calculateAssertionHash(predecessorId, afterState);
        require(!assertionExists(assertionHash), "Assertion already exists");
        require(assertionExists(predecessorId), "Predecessor does not exists");
        require(height > assertions[predecessorId].height, "Height too low");
        require(beforeStateHash == StateToolsLib.hash(assertions[predecessorId].state), "Before state hash does not match predecessor");

        return addAssertionUnsafe(predecessorId, height, nextInboxPosition, afterState, successionChallenge);
    }

    function setValidatorWhitelistDisabled(bool x) external {
        validatorWhitelistDisabled = x;
    }

    function setIsValidator(address user, bool x) external {
        isValidator[user] = x;
    }
}
