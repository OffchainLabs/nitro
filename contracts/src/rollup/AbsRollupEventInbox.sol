// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro-contracts/blob/main/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.0;

import "./IRollupEventInbox.sol";
import "../bridge/IBridge.sol";
import "../bridge/IEthBridge.sol";
import "../precompiles/ArbGasInfo.sol";
import "../libraries/ArbitrumChecker.sol";
import "../bridge/IDelayedMessageProvider.sol";
import "../libraries/DelegateCallAware.sol";
import {INITIALIZATION_MSG_TYPE} from "../libraries/MessageTypes.sol";
import {AlreadyInit, HadZeroInit, RollupNotChanged} from "../libraries/Error.sol";

/**
 * @title The inbox for rollup protocol events
 */
abstract contract AbsRollupEventInbox is
    IRollupEventInbox,
    IDelayedMessageProvider,
    DelegateCallAware
{
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

    /// @notice Allows the rollup owner to sync the rollup address
    function updateRollupAddress() external {
        if (msg.sender != IOwnable(rollup).owner())
            revert NotOwner(msg.sender, IOwnable(rollup).owner());
        address newRollup = address(bridge.rollup());
        if (rollup == newRollup) revert RollupNotChanged();
        rollup = newRollup;
    }

    function rollupInitialized(uint256 chainId, string calldata chainConfig)
        external
        override
        onlyRollup
    {
        require(bytes(chainConfig).length > 0, "EMPTY_CHAIN_CONFIG");
        uint8 initMsgVersion = 1;
        uint256 currentDataCost = block.basefee;
        if (ArbitrumChecker.runningOnArbitrum()) {
            currentDataCost += ArbGasInfo(address(0x6c)).getL1BaseFeeEstimate();
        }
        bytes memory initMsg = abi.encodePacked(
            chainId,
            initMsgVersion,
            currentDataCost,
            chainConfig
        );
        uint256 num = _enqueueInitializationMsg(initMsg);
        emit InboxMessageDelivered(num, initMsg);
    }

    function _enqueueInitializationMsg(bytes memory initMsg) internal virtual returns (uint256);
}
