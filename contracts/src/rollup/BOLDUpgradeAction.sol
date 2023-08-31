// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.0;

import "@openzeppelin/contracts-upgradeable/utils/Create2Upgradeable.sol";
import "@openzeppelin/contracts/proxy/transparent/ProxyAdmin.sol";
import "./RollupProxy.sol";
import "./RollupLib.sol";
import "./RollupAdminLogic.sol";

struct Node {
    // Hash of the state of the chain as of this node
    bytes32 stateHash;
    // Hash of the data that can be challenged
    bytes32 challengeHash;
    // Hash of the data that will be committed if this node is confirmed
    bytes32 confirmData;
    // Index of the node previous to this one
    uint64 prevNum;
    // Deadline at which this node can be confirmed
    uint64 deadlineBlock;
    // Deadline at which a child of this node can be confirmed
    uint64 noChildConfirmedBeforeBlock;
    // Number of stakers staked on this node. This includes real stakers and zombies
    uint64 stakerCount;
    // Number of stakers staked on a child node. This includes real stakers and zombies
    uint64 childStakerCount;
    // This value starts at zero and is set to a value when the first child is created. After that it is constant until the node is destroyed or the owner destroys pending nodes
    uint64 firstChildBlock;
    // The number of the latest child of this node to be created
    uint64 latestChildNumber;
    // The block number when this node was created
    uint64 createdAtBlock;
    // A hash of all the data needed to determine this node's validity, to protect against reorgs
    bytes32 nodeHash;
}

struct OldStaker {
    uint256 amountStaked;
    uint64 index;
    uint64 latestStakedNode;
    // currentChallenge is 0 if staker is not in a challenge
    uint64 currentChallenge; // 1. cannot have current challenge
    bool isStaked; // 2. must be staked
}

interface IOldRollup {
    struct Assertion {
        ExecutionState beforeState;
        ExecutionState afterState;
        uint64 numBlocks;
    }

    event NodeCreated(
        uint64 indexed nodeNum,
        bytes32 indexed parentNodeHash,
        bytes32 indexed nodeHash,
        bytes32 executionHash,
        Assertion assertion,
        bytes32 afterInboxBatchAcc,
        bytes32 wasmModuleRoot,
        uint256 inboxMaxCount
    );

    function wasmModuleRoot() external view returns (bytes32);
    function latestConfirmed() external view returns (uint64);
    function getNode(uint64 nodeNum) external view returns (Node memory);
    function getStakerAddress(uint64 stakerNum) external view returns (address);
    function stakerCount() external view returns (uint64);
    function getStaker(address staker) external view returns (OldStaker memory);
    function isValidator(address validator) external view returns (bool);
    function validatorWalletCreator() external view returns (address);
}

interface IOldRollupAdmin {
    function forceRefundStaker(address[] memory stacker) external;
    function pause() external;
    function resume() external;
}

/// @title  Provides pre-images to a state hash
/// @notice We need to use the execution state of the latest confirmed node as the genesis
///         in the new rollup. However the this full state is not available on chain, only
///         the state hash is, which commits to this. This lookup contract should be deployed
///         before the upgrade, and just before the upgrade is executed the pre-image of the
///         latest confirmed state hash should be populated here. The upgrade contact can then
///         fetch this information and verify it before using it.
contract StateHashPreImageLookup {
    using GlobalStateLib for GlobalState;

    event HashSet(bytes32 h, ExecutionState execState, uint256 inboxMaxCount);

    mapping(bytes32 => bytes) internal preImages;

    function stateHash(ExecutionState calldata execState, uint256 inboxMaxCount) public pure returns (bytes32) {
        return keccak256(abi.encodePacked(execState.globalState.hash(), inboxMaxCount, execState.machineStatus));
    }

    function set(bytes32 h, ExecutionState calldata execState, uint256 inboxMaxCount) public {
        require(h == stateHash(execState, inboxMaxCount), "Invalid hash");
        preImages[h] = abi.encode(execState, inboxMaxCount);
        emit HashSet(h, execState, inboxMaxCount);
    }

    function get(bytes32 h) public view returns (ExecutionState memory execState, uint256 inboxMaxCount) {
        (execState, inboxMaxCount) = abi.decode(preImages[h], (ExecutionState, uint256));
        require(inboxMaxCount != 0, "Hash not yet set");
    }
}

/// @title  Forwards calls to the rollup so that they can be interpreted as a user
/// @notice In the upgrade executor we need to access functions on the rollup
///         but since the upgrade executor is the admin it will always be forwarded to the
///         rollup admin logic. We create a separate forwarder contract here that just relays
///         information, since it's not the admin it can access rollup user logic.
contract RollupReader is IOldRollup {
    IOldRollup public immutable rollup;

    constructor(IOldRollup _rollup) {
        rollup = _rollup;
    }

    function wasmModuleRoot() external view returns (bytes32) {
        return rollup.wasmModuleRoot();
    }

    function latestConfirmed() external view returns (uint64) {
        return rollup.latestConfirmed();
    }

    function getNode(uint64 nodeNum) external view returns (Node memory) {
        return rollup.getNode(nodeNum);
    }

    function getStakerAddress(uint64 stakerNum) external view returns (address) {
        return rollup.getStakerAddress(stakerNum);
    }

    function stakerCount() external view returns (uint64) {
        return rollup.stakerCount();
    }

    function getStaker(address staker) external view returns (OldStaker memory) {
        return rollup.getStaker(staker);
    }

    function isValidator(address validator) external view returns (bool) {
        return rollup.isValidator(validator);
    }

    function validatorWalletCreator() external view returns (address) {
        return rollup.validatorWalletCreator();
    }
}

/// @title  Upgrades an Arbitrum rollup to the new challenge protocol
/// @notice Requires implementation contracts to be pre-deployed and provided in the constructor
///         Also requires a lookup contract to be provided that contains the pre-image of the state hash
///         that is in the latest confirmed assertion in the current rollup.
contract BOLDUpgradeAction {
    uint256 public constant BLOCK_LEAF_SIZE = 2 ** 26;
    uint256 public constant BIGSTEP_LEAF_SIZE = 2 ** 23;
    uint256 public constant SMALLSTEP_LEAF_SIZE = 2 ** 20;

    address public immutable L1_TIMELOCK;
    IOldRollup public immutable OLD_ROLLUP;
    address public immutable BRIDGE;
    address public immutable SEQ_INBOX;
    address public immutable REI;
    address public immutable OUTBOX;
    address public immutable INBOX;

    uint64 public immutable CONFIRM_PERIOD_BLOCKS;
    address public immutable STAKE_TOKEN;
    uint256 public immutable STAKE_AMOUNT;
    uint256 public immutable MINI_STAKE_AMOUNT;
    uint256 public immutable CHAIN_ID;
    address public immutable ANY_TRUST_FAST_CONFIRMER;
    bool public immutable DISABLE_VALIDATOR_WHITELIST;

    IOneStepProofEntry public immutable OSP;
    // proxy admins of the contracts to be upgraded
    ProxyAdmin public immutable PROXY_ADMIN_OUTBOX;
    ProxyAdmin public immutable PROXY_ADMIN_BRIDGE;
    ProxyAdmin public immutable PROXY_ADMIN_REI;
    ProxyAdmin public immutable PROXY_ADMIN_SEQUENCER_INBOX;
    StateHashPreImageLookup public immutable PREIMAGE_LOOKUP;
    RollupReader public immutable ROLLUP_READER;

    // new contract implementations
    address public immutable IMPL_BRIDGE;
    address public immutable IMPL_SEQUENCER_INBOX;
    address public immutable IMPL_REI;
    address public immutable IMPL_OUTBOX;
    // the old rollup, but with whenNotPaused protection removed from stake withdrawal functions
    address public immutable IMPL_PATCHED_OLD_ROLLUP_USER;
    address public immutable IMPL_NEW_ROLLUP_USER;
    address public immutable IMPL_NEW_ROLLUP_ADMIN;
    address public immutable IMPL_CHALLENGE_MANAGER;

    event RollupMigrated(address rollup, address challengeManager);

    struct Settings {
        uint64 confirmPeriodBlocks;
        address stakeToken;
        uint256 stakeAmt;
        uint256 miniStakeAmt;
        uint256 chainId;
        address anyTrustFastConfirmer;
        bool disableValidatorWhitelist;
    }

    // Unfortunately these are not discoverable on-chain, so we need to supply them
    struct ProxyAdmins {
        address outbox;
        address bridge;
        address rei;
        address seqInbox;
    }

    struct Implementations {
        address bridge;
        address seqInbox;
        address rei;
        address outbox;
        address oldRollupUser;
        address newRollupUser;
        address newRollupAdmin;
        address challengeManager;
    }

    struct Contracts {
        address l1Timelock;
        IOldRollup rollup;
        address bridge;
        address sequencerInbox;
        address rollupEventInbox;
        address outbox;
        address inbox;
        IOneStepProofEntry osp;
    }

    constructor(
        Contracts memory contracts,
        ProxyAdmins memory proxyAdmins,
        Implementations memory implementations,
        Settings memory settings
    ) {
        L1_TIMELOCK = contracts.l1Timelock;
        OLD_ROLLUP = contracts.rollup;
        BRIDGE = contracts.bridge;
        SEQ_INBOX = contracts.sequencerInbox;
        REI = contracts.rollupEventInbox;
        OUTBOX = contracts.outbox;
        INBOX = contracts.inbox;
        OSP = contracts.osp;

        PROXY_ADMIN_OUTBOX = ProxyAdmin(proxyAdmins.outbox);
        PROXY_ADMIN_BRIDGE = ProxyAdmin(proxyAdmins.bridge);
        PROXY_ADMIN_REI = ProxyAdmin(proxyAdmins.rei);
        PROXY_ADMIN_SEQUENCER_INBOX = ProxyAdmin(proxyAdmins.seqInbox);
        PREIMAGE_LOOKUP = new StateHashPreImageLookup();
        ROLLUP_READER = new RollupReader(contracts.rollup);

        IMPL_BRIDGE = implementations.bridge;
        IMPL_SEQUENCER_INBOX = implementations.seqInbox;
        IMPL_REI = implementations.rei;
        IMPL_OUTBOX = implementations.outbox;
        IMPL_PATCHED_OLD_ROLLUP_USER = implementations.oldRollupUser;
        IMPL_NEW_ROLLUP_USER = implementations.newRollupUser;
        IMPL_NEW_ROLLUP_ADMIN = implementations.newRollupAdmin;
        IMPL_CHALLENGE_MANAGER = implementations.challengeManager;

        CHAIN_ID = settings.chainId;
        CONFIRM_PERIOD_BLOCKS = settings.confirmPeriodBlocks;
        STAKE_TOKEN = settings.stakeToken;
        STAKE_AMOUNT = settings.stakeAmt;
        MINI_STAKE_AMOUNT = settings.miniStakeAmt;
        ANY_TRUST_FAST_CONFIRMER = settings.anyTrustFastConfirmer;
        DISABLE_VALIDATOR_WHITELIST = settings.disableValidatorWhitelist;
    }

    /// @dev    Refund the existing stakers, pause and upgrade the current rollup to
    ///         allow them to withdraw after pausing
    function cleanupOldRollup() private {
        IOldRollupAdmin(address(OLD_ROLLUP)).pause();

        uint64 stakerCount = ROLLUP_READER.stakerCount();
        // since we for-loop these stakers we set an arbitrary limit - we dont
        // expect any instances to have close to this number of stakers
        if (stakerCount > 50) {
            stakerCount = 50;
        }
        for (uint64 i = 0; i < stakerCount; i++) {
            address stakerAddr = ROLLUP_READER.getStakerAddress(i);
            OldStaker memory staker = ROLLUP_READER.getStaker(stakerAddr);
            if (staker.isStaked && staker.currentChallenge == 0) {
                address[] memory stakersToRefund = new address[](1);
                stakersToRefund[0] = stakerAddr;

                IOldRollupAdmin(address(OLD_ROLLUP)).forceRefundStaker(stakersToRefund);
            }
        }

        // upgrade the rollup to one that allows validators to withdraw even whilst paused
        DoubleLogicUUPSUpgradeable(address(OLD_ROLLUP)).upgradeSecondaryTo(IMPL_PATCHED_OLD_ROLLUP_USER);
    }

    /// @dev    Create a config for the new rollup - fetches the latest confirmed
    ///         assertion from the old rollup and uses it as genesis
    function createConfig() private view returns (Config memory) {
        // fetch the assertion associated with the latest confirmed state
        bytes32 latestConfirmedStateHash = ROLLUP_READER.getNode(ROLLUP_READER.latestConfirmed()).stateHash;
        (ExecutionState memory genesisExecState, uint256 inboxMaxCount) = PREIMAGE_LOOKUP.get(latestConfirmedStateHash);
        // double check the hash
        require(
            RollupLib.stateHashMem(genesisExecState, inboxMaxCount) == latestConfirmedStateHash,
            "Invalid latest execution hash"
        );

        // this isnt used during rollup creation, so we can pass in empty
        ISequencerInbox.MaxTimeVariation memory maxTimeVariation;

        return Config({
            confirmPeriodBlocks: CONFIRM_PERIOD_BLOCKS,
            stakeToken: STAKE_TOKEN,
            baseStake: STAKE_AMOUNT,
            wasmModuleRoot: ROLLUP_READER.wasmModuleRoot(),
            owner: address(this), // upgrade executor is the owner
            loserStakeEscrow: L1_TIMELOCK, // additional funds get sent to the l1 timelock
            chainId: CHAIN_ID,
            chainConfig: "", // we can use an empty chain config it wont be used in the rollup initialization because we check if the rei is already connected there
            miniStakeValue: MINI_STAKE_AMOUNT,
            sequencerInboxMaxTimeVariation: maxTimeVariation,
            layerZeroBlockEdgeHeight: BLOCK_LEAF_SIZE,
            layerZeroBigStepEdgeHeight: BIGSTEP_LEAF_SIZE,
            layerZeroSmallStepEdgeHeight: SMALLSTEP_LEAF_SIZE,
            genesisExecutionState: genesisExecState,
            genesisInboxCount: inboxMaxCount,
            anyTrustFastConfirmer: ANY_TRUST_FAST_CONFIRMER
        });
    }

    function upgradeSurroundingContracts(address newRollupAddress) private {
        // now we upgrade each of the contracts that a reference to the rollup address
        // first we upgrade to an implementation which allows setting, then set the rollup address
        // then we revert to the previous implementation since we dont require this functionality going forward

        TransparentUpgradeableProxy bridge = TransparentUpgradeableProxy(payable(BRIDGE));
        address currentBridgeImpl = PROXY_ADMIN_BRIDGE.getProxyImplementation(bridge);
        PROXY_ADMIN_BRIDGE.upgradeAndCall(
            bridge, IMPL_BRIDGE, abi.encodeWithSelector(IBridge.updateRollupAddress.selector, newRollupAddress)
        );
        PROXY_ADMIN_BRIDGE.upgrade(bridge, currentBridgeImpl);

        TransparentUpgradeableProxy sequencerInbox = TransparentUpgradeableProxy(payable(SEQ_INBOX));
        address currentSequencerInboxImpl = PROXY_ADMIN_BRIDGE.getProxyImplementation(sequencerInbox);
        PROXY_ADMIN_SEQUENCER_INBOX.upgradeAndCall(
            sequencerInbox, IMPL_SEQUENCER_INBOX, abi.encodeWithSelector(IOutbox.updateRollupAddress.selector)
        );
        PROXY_ADMIN_SEQUENCER_INBOX.upgrade(sequencerInbox, currentSequencerInboxImpl);

        TransparentUpgradeableProxy rollupEventInbox = TransparentUpgradeableProxy(payable(REI));
        address currentRollupEventInboxImpl = PROXY_ADMIN_REI.getProxyImplementation(rollupEventInbox);
        PROXY_ADMIN_REI.upgradeAndCall(
            rollupEventInbox, IMPL_REI, abi.encodeWithSelector(IOutbox.updateRollupAddress.selector)
        );
        PROXY_ADMIN_REI.upgrade(rollupEventInbox, currentRollupEventInboxImpl);

        TransparentUpgradeableProxy outbox = TransparentUpgradeableProxy(payable(OUTBOX));
        address currentOutboxImpl = PROXY_ADMIN_REI.getProxyImplementation(outbox);
        PROXY_ADMIN_OUTBOX.upgradeAndCall(
            outbox, IMPL_OUTBOX, abi.encodeWithSelector(IOutbox.updateRollupAddress.selector)
        );
        PROXY_ADMIN_OUTBOX.upgrade(outbox, currentOutboxImpl);
    }

    function perform(address[] memory validators) external {
        // tidy up the old rollup - pause it and refund stakes
        cleanupOldRollup();

        // create the config, we do this now so that we compute the expected rollup address
        Config memory config = createConfig();

        // deploy the new challenge manager
        IEdgeChallengeManager challengeManager = IEdgeChallengeManager(
            address(
                new TransparentUpgradeableProxy(
                    address(IMPL_CHALLENGE_MANAGER),
                    address(PROXY_ADMIN_BRIDGE), // use the same proxy admin as the bridge
                    ""
                )
            )
        );

        // now that all the dependent contracts are pointed at the new address we can
        // deploy and init the new rollup
        ContractDependencies memory connectedContracts = ContractDependencies({
            bridge: IBridge(BRIDGE),
            sequencerInbox: ISequencerInbox(SEQ_INBOX),
            inbox: IInbox(INBOX),
            outbox: IOutbox(OUTBOX),
            rollupEventInbox: IRollupEventInbox(REI),
            challengeManager: challengeManager,
            rollupAdminLogic: IMPL_NEW_ROLLUP_ADMIN,
            rollupUserLogic: IRollupUser(IMPL_NEW_ROLLUP_USER),
            validatorWalletCreator: ROLLUP_READER.validatorWalletCreator()
        });

        // upgrade the surrounding contracts eg bridge, outbox, seq inbox, rollup event inbox
        // to set of the new rollup address
        bytes32 rollupSalt = keccak256(abi.encode(config));
        address expectedRollupAddress =
            Create2Upgradeable.computeAddress(rollupSalt, keccak256(type(RollupProxy).creationCode));
        upgradeSurroundingContracts(expectedRollupAddress);

        challengeManager.initialize({
            _assertionChain: IAssertionChain(expectedRollupAddress),
            // confirm period and challenge period are the same atm
            _challengePeriodBlocks: config.confirmPeriodBlocks,
            _oneStepProofEntry: OSP,
            layerZeroBlockEdgeHeight: config.layerZeroBlockEdgeHeight,
            layerZeroBigStepEdgeHeight: config.layerZeroBigStepEdgeHeight,
            layerZeroSmallStepEdgeHeight: config.layerZeroSmallStepEdgeHeight,
            _stakeToken: IERC20(config.stakeToken),
            _stakeAmount: config.miniStakeValue,
            _excessStakeReceiver: L1_TIMELOCK
        });

        RollupProxy rollup = new RollupProxy{ salt: rollupSalt}();
        require(address(rollup) == expectedRollupAddress, "UNEXPCTED_ROLLUP_ADDR");

        // initialize the rollup with this contract as owner to set batch poster and validators
        // it will transfer the ownership back to the actual owner later
        address actualOwner = config.owner;
        config.owner = address(this);

        rollup.initializeProxy(config, connectedContracts);

        if (validators.length != 0) {
            bool[] memory _vals = new bool[](validators.length);
            for (uint256 i = 0; i < validators.length; i++) {
                require(ROLLUP_READER.isValidator(validators[i]), "UNEXPECTED_NEW_VALIDATOR");
                _vals[i] = true;
            }
            IRollupAdmin(address(rollup)).setValidator(validators, _vals);
        }
        if (DISABLE_VALIDATOR_WHITELIST) {
            IRollupAdmin(address(rollup)).setValidatorWhitelistDisabled(DISABLE_VALIDATOR_WHITELIST);
        }

        IRollupAdmin(address(rollup)).setOwner(actualOwner);

        emit RollupMigrated(expectedRollupAddress, address(challengeManager));
    }
}
