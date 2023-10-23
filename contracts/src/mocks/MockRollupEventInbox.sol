// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.0;

import "../rollup/IRollupEventInbox.sol";
import "../bridge/IBridge.sol";
import "../bridge/IDelayedMessageProvider.sol";
import "../precompiles/ArbGasInfo.sol";
import "../libraries/DelegateCallAware.sol";
import "../libraries/ArbitrumChecker.sol";
import {INITIALIZATION_MSG_TYPE} from "../libraries/MessageTypes.sol";
import {AlreadyInit, HadZeroInit} from "../libraries/Error.sol";

/**
 * @title The inbox for rollup protocol events
 */
contract MockRollupEventInbox is IRollupEventInbox, IDelayedMessageProvider, DelegateCallAware {
    IBridge public override bridge;
    address public override rollup;

    modifier onlyRollup() {
        require(msg.sender == rollup, "ONLY_ROLLUP");
        _;
    }

    function initialize(IBridge _bridge) external override onlyDelegated {
        if (address(bridge) != address(0)) revert AlreadyInit();
        if (address(_bridge) == address(0)) revert HadZeroInit();
        bridge = _bridge;
        rollup = address(_bridge.rollup());
    }

    /// @notice Allows the proxy owner to set the rollup address
    function updateRollupAddress() external onlyDelegated onlyProxyOwner {
        rollup = address(bridge.rollup());
    }

    function rollupInitialized(uint256 chainId, string calldata chainConfig)
        external
        override
        onlyRollup
    {
        require(bytes(chainConfig).length > 0, "EMPTY_CHAIN_CONFIG");
        uint8 initMsgVersion = 1;
        uint256 currentDataCost = 1; // Set to a base fee of 1.
        if (ArbitrumChecker.runningOnArbitrum()) {
            currentDataCost += ArbGasInfo(address(0x6c)).getL1BaseFeeEstimate();
        }
        bytes memory initMsg = abi.encodePacked(
            chainId,
            initMsgVersion,
            currentDataCost,
            chainConfig
        );
        uint256 num = bridge.enqueueDelayedMessage(
            INITIALIZATION_MSG_TYPE,
            address(0),
            keccak256(initMsg)
        );
        emit InboxMessageDelivered(num, initMsg);
    }
}
