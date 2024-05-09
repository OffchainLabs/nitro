// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro-contracts/blob/main/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.0;

import "../state/GlobalState.sol";
import "../state/Machine.sol";
import "../bridge/ISequencerInbox.sol";
import "../bridge/IBridge.sol";
import "../bridge/IOutbox.sol";
import "../bridge/IInboxBase.sol";
import "./IRollupEventInbox.sol";
import "./IRollupLogic.sol";
import "../challengeV2/EdgeChallengeManager.sol";

struct Config {
    uint64 confirmPeriodBlocks;
    address stakeToken;
    uint256 baseStake;
    bytes32 wasmModuleRoot;
    address owner;
    address loserStakeEscrow;
    uint256 chainId;
    string chainConfig;
    uint256[] miniStakeValues;
    ISequencerInbox.MaxTimeVariation sequencerInboxMaxTimeVariation;
    uint256 layerZeroBlockEdgeHeight;
    uint256 layerZeroBigStepEdgeHeight;
    uint256 layerZeroSmallStepEdgeHeight;
    /// @notice The execution state to be used in the genesis assertion
    AssertionState genesisAssertionState;
    /// @notice The inbox size at the time the genesis execution state was created
    uint256 genesisInboxCount;
    address anyTrustFastConfirmer;
    uint8 numBigStepLevel;
    uint64 challengeGracePeriodBlocks;
    BufferConfig bufferConfig;
}

struct ContractDependencies {
    IBridge bridge;
    ISequencerInbox sequencerInbox;
    IInboxBase inbox;
    IOutbox outbox;
    IRollupEventInbox rollupEventInbox;
    IEdgeChallengeManager challengeManager;
    address rollupAdminLogic; // this cannot be IRollupAdmin because of circular dependencies
    IRollupUser rollupUserLogic;
    address validatorWalletCreator;
}
