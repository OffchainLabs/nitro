// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.4;

import "@openzeppelin/contracts-upgradeable/proxy/utils/Initializable.sol";
import "@openzeppelin/contracts-upgradeable/utils/AddressUpgradeable.sol";

import {
    NotContract,
    NotRollupOrOwner,
    NotDelayedInbox,
    NotSequencerInbox,
    NotOutbox,
    InvalidOutboxSet,
    BadSequencerMessageNumber
} from "../libraries/Error.sol";
import "./AbsBridge.sol";
import "./IEthBridge.sol";
import "./Messages.sol";
import "../libraries/DelegateCallAware.sol";

import {L1MessageType_batchPostingReport} from "../libraries/MessageTypes.sol";

/**
 * @title Staging ground for incoming and outgoing messages
 * @notice It is also the ETH escrow for value sent with these messages.
 */
contract Bridge is AbsBridge, IEthBridge {
    using AddressUpgradeable for address;

    /// @inheritdoc IEthBridge
    function initialize(IOwnable rollup_) external initializer onlyDelegated {
        _activeOutbox = EMPTY_ACTIVEOUTBOX;
        rollup = rollup_;
    }

    /// @inheritdoc IEthBridge
    function enqueueDelayedMessage(
        uint8 kind,
        address sender,
        bytes32 messageDataHash
    ) external payable virtual returns (uint256) {
        if (!allowedDelayedInboxes(msg.sender)) revert NotDelayedInbox(msg.sender);
        return
            addMessageToDelayedAccumulator(
                kind,
                sender,
                uint64(block.number),
                uint64(block.timestamp), // solhint-disable-line not-rely-on-time
                block.basefee,
                messageDataHash
            );
    }

    /// @inheritdoc IBridge
    function executeCall(
        address to,
        uint256 value,
        bytes calldata data
    ) external returns (bool success, bytes memory returnData) {
        if (!allowedOutboxes(msg.sender)) revert NotOutbox(msg.sender);
        if (data.length > 0 && !to.isContract()) revert NotContract(to);
        address prevOutbox = _activeOutbox;
        _activeOutbox = msg.sender;
        // We set and reset active outbox around external call so activeOutbox remains valid during call

        // We use a low level call here since we want to bubble up whether it succeeded or failed to the caller
        // rather than reverting on failure as well as allow contract and non-contract calls
        // solhint-disable-next-line avoid-low-level-calls
        (success, returnData) = to.call{value: value}(data);
        _activeOutbox = prevOutbox;
        emit BridgeCallTriggered(msg.sender, to, value, data);
    }
}
