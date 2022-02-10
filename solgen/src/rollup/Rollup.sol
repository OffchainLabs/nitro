// SPDX-License-Identifier: Apache-2.0

/*
 * Copyright 2021, Offchain Labs, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

pragma solidity ^0.8.0;

import "@openzeppelin/contracts/proxy/Proxy.sol";
import "@openzeppelin/contracts/utils/Address.sol";

import "./RollupCore.sol";
import "./RollupEventBridge.sol";
import "./RollupLib.sol";
import "./Node.sol";

import "../libraries/ProxyUtil.sol";

contract Rollup is Proxy, RollupCore {
    using Address for address;

    function isInit() internal view returns (bool) {
        return confirmPeriodBlocks != 0 || isMasterCopy;
    }

    struct ContractDependencies {
        IBridge delayedBridge;
        ISequencerInbox sequencerInbox;
        IOutbox outbox;
        RollupEventBridge rollupEventBridge;
        IBlockChallengeFactory blockChallengeFactory;

        IRollupAdmin rollupAdminLogic;
        IRollupUser rollupUserLogic;
    }

    // _rollupParams = [ confirmPeriodBlocks, extraChallengeTimeBlocks, chainId, baseStake ]
    // connectedContracts = [delayedBridge, sequencerInbox, outbox, rollupEventBridge, blockChallengeFactory]
    // sequencerInboxParams = [ maxDelayBlocks, maxFutureBlocks, maxDelaySeconds, maxFutureSeconds ]
    function initialize(
        RollupLib.Config memory config,
        ContractDependencies memory connectedContracts
    ) external {
        require(!isInit(), "ALREADY_INIT");

        // calls initialize method in user logic
        require(address(connectedContracts.rollupAdminLogic).isContract(), "ADMIN_LOGIC_NOT_CONTRACT");
        require(address(connectedContracts.rollupUserLogic).isContract(), "USER_LOGIC_NOT_CONTRACT");
        (bool success, ) = address(connectedContracts.rollupUserLogic).delegatecall(
            abi.encodeWithSelector(IRollupUser.initialize.selector, config.stakeToken)
        );
        adminLogic = connectedContracts.rollupAdminLogic;
        userLogic = connectedContracts.rollupUserLogic;
        require(success, "FAIL_INIT_LOGIC");

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
        owner = config.owner;
        wasmModuleRoot = config.wasmModuleRoot;
        // A little over 15 minutes
        minimumAssertionPeriod = 75;
        challengeExecutionBisectionDegree = 400;

        sequencerBridge.setMaxTimeVariation(config.sequencerInboxMaxTimeVariation);

        emit RollupInitialized(config.wasmModuleRoot, config.chainId);
        require(isInit(), "INITIALIZE_NOT_INIT");
    }

    function postUpgradeInit() external {
        // it is assumed the rollup contract is behind a Proxy controlled by a proxy admin
        // this function can only be called by the proxy admin contract
        address proxyAdmin = ProxyUtil.getProxyAdmin();
        require(msg.sender == proxyAdmin, "NOT_FROM_ADMIN");
    }

    function createInitialNode()
        private
        view
        returns (Node memory)
    {
        GlobalState memory emptyGlobalState;
        bytes32 state = RollupLib.stateHash(
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
     * @dev This is a virtual function that should be overriden so it returns the address to which the fallback function
     * and {_fallback} should delegate.
     */
    function _implementation()
        internal
        view
        virtual
        override
        returns (address)
    {
        require(msg.data.length >= 4, "NO_FUNC_SIG");
        address rollupOwner = owner;
        // if there is an owner and it is the sender, delegate to admin logic
        address target = rollupOwner != address(0) && rollupOwner == msg.sender
            ? address(adminLogic)
            : address(userLogic);
        require(target.isContract(), "TARGET_NOT_CONTRACT");
        return target;
    }
}
