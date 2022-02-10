// SPDX-License-Identifier: Apache-2.0

pragma solidity ^0.8.0;

import { AAPStorage } from  "./AdminAwareProxy.sol";
import { IRollupAdmin } from "./IRollupLogic.sol";
import "./RollupCore.sol";
import "../bridge/IOutbox.sol";
import "../bridge/ISequencerInbox.sol";
import "../challenge/IChallenge.sol";

import "@openzeppelin/contracts/proxy/beacon/UpgradeableBeacon.sol";

contract RollupAdminLogic is AAPStorage, RollupCore, IRollupAdmin {
    function isInit() internal view returns (bool) {
        return confirmPeriodBlocks != 0 || isMasterCopy;
    }

    function initialize(
        RollupLib.Config calldata config,
        ContractDependencies calldata connectedContracts
    ) external override {
        require(!isInit(), "NOT_INIT");

        delayedBridge = connectedContracts.delayedBridge;
        sequencerBridge = connectedContracts.sequencerInbox;
        outbox = connectedContracts.outbox;
        delayedBridge.setOutbox(address(connectedContracts.outbox), true);
        rollupEventBridge = connectedContracts.rollupEventBridge;
        delayedBridge.setInbox(address(connectedContracts.rollupEventBridge), true);

        rollupEventBridge.rollupInitialized(config.owner, config.chainId);
        sequencerBridge.addSequencerL2Batch(0, "", 1, IGasRefunder(address(0)));

        challengeFactory = connectedContracts.blockChallengeFactory;

        Node memory node = createInitialNode();
        initializeCore(node);

        confirmPeriodBlocks = config.confirmPeriodBlocks;
        extraChallengeTimeBlocks = config.extraChallengeTimeBlocks;
        chainId = config.chainId;
        baseStake = config.baseStake;
        wasmModuleRoot = config.wasmModuleRoot;
        // A little over 15 minutes
        minimumAssertionPeriod = 75;
        challengeExecutionBisectionDegree = 400;

        sequencerBridge.setMaxTimeVariation(config.sequencerInboxMaxTimeVariation);

        emit RollupInitialized(config.wasmModuleRoot, config.chainId);
        require(isInit(), "INITIALIZE_NOT_INIT");
    }

    function createInitialNode()
        private
        view
        returns (Node memory)
    {
        GlobalState memory emptyGlobalState;
        bytes32 state = RollupLib.stateHashMem(
            RollupLib.ExecutionState(
                emptyGlobalState,
                1, // inboxMaxCount - force the first assertion to read a message
                MachineStatus.FINISHED
            )
        );
        return
            NodeLib.initialize(
                state,
                0, // challenge hash (not challengeable)
                0, // confirm data
                0, // prev node
                uint64(block.number), // deadline block (not challengeable)
                0 // initial node has a node hash of 0
            );
    }

    /**
     * Functions are only to reach this logic contract if the caller is the owner
     * so there is no need for a redundant onlyOwner check
     */

    /**
     * @notice Add a contract authorized to put messages into this rollup's inbox
     * @param _outbox Outbox contract to add
     */
    function setOutbox(IOutbox _outbox) external override {
        outbox = _outbox;
        delayedBridge.setOutbox(address(_outbox), true);
        emit OwnerFunctionCalled(0);
    }

    /**
     * @notice Disable an old outbox from interacting with the bridge
     * @param _outbox Outbox contract to remove
     */
    function removeOldOutbox(address _outbox) external override {
        require(_outbox != address(outbox), "CUR_OUTBOX");
        delayedBridge.setOutbox(_outbox, false);
        emit OwnerFunctionCalled(1);
    }

    /**
     * @notice Enable or disable an inbox contract
     * @param _inbox Inbox contract to add or remove
     * @param _enabled New status of inbox
     */
    function setInbox(address _inbox, bool _enabled) external override {
        delayedBridge.setInbox(address(_inbox), _enabled);
        emit OwnerFunctionCalled(2);
    }

    /**
     * @notice Pause interaction with the rollup contract.
     * The time spent paused is not incremented in the rollup's timing for node validation.
     */
    function pause() external override {
        _pause();
        emit OwnerFunctionCalled(3);
    }

    /**
     * @notice Resume interaction with the rollup contract
     */
    function resume() external override {
        _unpause();
        emit OwnerFunctionCalled(4);
    }

    /**
     * @notice Set the addresses of rollup logic contracts called
     * @param newAdminLogic address of logic that owner of rollup calls
     * @param newUserLogic address of logic that user of rollup calls
     */
    function setLogicContracts(address newAdminLogic, address newUserLogic) external override {
        adminLogic = AAPStorage(newAdminLogic);
        userLogic = AAPStorage(newUserLogic);
        emit OwnerFunctionCalled(5);
    }

    /**
     * @notice Set the addresses of the validator whitelist
     * @dev It is expected that both arrays are same length, and validator at
     * position i corresponds to the value at position i
     * @param _validator addresses to set in the whitelist
     * @param _val value to set in the whitelist for corresponding address
     */
    function setValidator(address[] calldata _validator, bool[] calldata _val) external override {
        require(_validator.length == _val.length, "WRONG_LENGTH");

        for (uint256 i = 0; i < _validator.length; i++) {
            isValidator[_validator[i]] = _val[i];
        }
        emit OwnerFunctionCalled(6);
    }

    /**
     * @notice Set a new owner address for the rollup
     * @param newOwner address of new rollup owner
     */
    function setOwner(address newOwner) external override {
        owner = newOwner;
        emit OwnerFunctionCalled(7);
    }

    /**
     * @notice Set minimum assertion period for the rollup
     * @param newPeriod new minimum period for assertions
     */
    function setMinimumAssertionPeriod(uint256 newPeriod) external override {
        minimumAssertionPeriod = newPeriod;
        emit OwnerFunctionCalled(8);
    }

    /**
     * @notice Set number of blocks until a node is considered confirmed
     * @param newConfirmPeriod new number of blocks
     */
    function setConfirmPeriodBlocks(uint64 newConfirmPeriod) external override {
        confirmPeriodBlocks = newConfirmPeriod;
        emit OwnerFunctionCalled(9);
    }

    /**
     * @notice Set number of extra blocks after a challenge
     * @param newExtraTimeBlocks new number of blocks
     */
    function setExtraChallengeTimeBlocks(uint64 newExtraTimeBlocks) external override {
        extraChallengeTimeBlocks = newExtraTimeBlocks;
        emit OwnerFunctionCalled(10);
    }

    /**
     * @notice Set the proving WASM module root
     * @param newWasmModuleRoot new module root
     */
    function setWasmModuleRoot(bytes32 newWasmModuleRoot) external override {
        wasmModuleRoot = newWasmModuleRoot;
        emit OwnerFunctionCalled(11);
    }

    /**
     * @notice Set base stake required for an assertion
     * @param newBaseStake minimum amount of stake required
     */
    function setBaseStake(uint256 newBaseStake) external override {
        baseStake = newBaseStake;
        emit OwnerFunctionCalled(12);
    }

    /**
     * @notice Set the token used for stake, where address(0) == eth
     * @dev Before changing the base stake token, you might need to change the
     * implementation of the Rollup User facet!
     * @param newStakeToken address of token used for staking
     */
    function setStakeToken(address newStakeToken) external override {
        stakeToken = newStakeToken;
        emit OwnerFunctionCalled(13);
    }

    /**
     * @notice Set max delay for sequencer inbox
     * @param maxTimeVariation the maximum time variation parameters
     */
    function setSequencerInboxMaxTimeVariation(ISequencerInbox.MaxTimeVariation calldata maxTimeVariation) external override {
        sequencerBridge.setMaxTimeVariation(maxTimeVariation);
        emit OwnerFunctionCalled(14);
    }

    /**
     * @notice Set execution bisection degree
     * @param newChallengeExecutionBisectionDegree execution bisection degree
     */
    function setChallengeExecutionBisectionDegree(uint256 newChallengeExecutionBisectionDegree)
        external
        override
    {
        challengeExecutionBisectionDegree = newChallengeExecutionBisectionDegree;
        emit OwnerFunctionCalled(16);
    }

    /**
     * @notice Updates whether an address is authorized to be a batch poster at the sequencer inbox
     * @param addr the address
     * @param isBatchPoster if the specified address should be authorized as a batch poster
     */
    function setIsBatchPoster(address addr, bool isBatchPoster) external override {
        ISequencerInbox(sequencerBridge).setIsBatchPoster(addr, isBatchPoster);
        emit OwnerFunctionCalled(19);
    }

    /**
     * @notice Upgrades the implementation of a beacon controlled by the rollup
     * @param beacon address of beacon to be upgraded
     * @param newImplementation new address of implementation
     */
    function upgradeBeacon(address beacon, address newImplementation) external override {
        UpgradeableBeacon(beacon).upgradeTo(newImplementation);
        emit OwnerFunctionCalled(20);
    }

    function forceResolveChallenge(address[] calldata stakerA, address[] calldata stakerB)
        external
        override
        whenPaused
    {
        require(stakerA.length == stakerB.length, "WRONG_LENGTH");
        for (uint256 i = 0; i < stakerA.length; i++) {
            IChallenge chall = inChallenge(stakerA[i], stakerB[i]);

            require(address(0) != address(chall), "NOT_IN_CHALL");
            clearChallenge(stakerA[i]);
            clearChallenge(stakerB[i]);

            chall.clearChallenge();
        }
        emit OwnerFunctionCalled(21);
    }

    function forceRefundStaker(address[] calldata staker) external override whenPaused {
        for (uint256 i = 0; i < staker.length; i++) {
            reduceStakeTo(staker[i], 0);
            turnIntoZombie(staker[i]);
        }
        emit OwnerFunctionCalled(22);
    }

    function forceCreateNode(
        uint64 prevNode,
        RollupLib.Assertion calldata assertion,
        bytes32 expectedNodeHash
    ) external override whenPaused {
        require(prevNode == latestConfirmed(), "ONLY_LATEST_CONFIRMED");

        createNewNode(
            assertion,
            prevNode,
            expectedNodeHash
        );

        emit OwnerFunctionCalled(23);
    }

    function forceConfirmNode(
        uint64 nodeNum,
        bytes32 blockHash,
        bytes32 sendRoot
    ) external override whenPaused {
        // this skips deadline, staker and zombie validation
        confirmNode(
            nodeNum,
            blockHash,
            sendRoot
        );
        emit OwnerFunctionCalled(24);
    }
}
