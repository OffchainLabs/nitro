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

    constructor(uint64 _confirmPeriodBlocks) Cloneable() Pausable() {
        // constructor is used so logic contract can't be init'ed
        confirmPeriodBlocks = _confirmPeriodBlocks;
        require(isInit(), "CONSTRUCTOR_NOT_INIT");
    }

    function isInit() internal view returns (bool) {
        return confirmPeriodBlocks != 0;
    }

    // _rollupParams = [ confirmPeriodBlocks, extraChallengeTimeBlocks, chainId, baseStake ]
    // connectedContracts = [delayedBridge, sequencerInbox, outbox, rollupEventBridge, blockChallengeFactory]
    // sequencerInboxParams = [ maxDelayBlocks, maxFutureBlocks, maxDelaySeconds, maxFutureSeconds ]
    function initialize(
        bytes32 _wasmModuleRoot,
        uint256[4] calldata _rollupParams,
        address _stakeToken,
        address _owner,
        address[5] calldata connectedContracts,
        address[2] calldata _logicContracts,
        uint256[4] calldata sequencerInboxParams
    ) public {
        require(!isInit(), "ALREADY_INIT");

        // calls initialize method in user logic
        require(_logicContracts[0].isContract(), "LOGIC_0_NOT_CONTRACT");
        require(_logicContracts[1].isContract(), "LOGIC_1_NOT_CONTRACT");
        (bool success, ) = _logicContracts[1].delegatecall(
            abi.encodeWithSelector(IRollupUser.initialize.selector, _stakeToken)
        );
        adminLogic = IRollupAdmin(_logicContracts[0]);
        userLogic = IRollupUser(_logicContracts[1]);
        require(success, "FAIL_INIT_LOGIC");

        delayedBridge = IBridge(connectedContracts[0]);
        sequencerBridge = ISequencerInbox(connectedContracts[1]);
        outbox = IOutbox(connectedContracts[2]);
        delayedBridge.setOutbox(connectedContracts[2], true);
        rollupEventBridge = RollupEventBridge(connectedContracts[3]);
        delayedBridge.setInbox(connectedContracts[3], true);

        rollupEventBridge.rollupInitialized(_owner, _rollupParams[2]);
        sequencerBridge.addSequencerL2Batch(0, "", 1, IGasRefunder(address(0)));

        challengeFactory = IBlockChallengeFactory(connectedContracts[4]);

        Node memory node = createInitialNode();
        initializeCore(node);

        confirmPeriodBlocks = uint64(_rollupParams[0]);
        extraChallengeTimeBlocks = uint64(_rollupParams[1]);
        chainId = _rollupParams[2];
        baseStake = _rollupParams[3];
        owner = _owner;
        wasmModuleRoot = _wasmModuleRoot;
        // A little over 15 minutes
        minimumAssertionPeriod = 75;
        challengeExecutionBisectionDegree = 400;

        sequencerBridge.setMaxTimeVariation(
            sequencerInboxParams[0],
            sequencerInboxParams[1],
            sequencerInboxParams[2],
            sequencerInboxParams[3]
        );

        emit RollupCreated(_wasmModuleRoot);
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
                uint64(block.number) // deadline block (not challengeable)
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
