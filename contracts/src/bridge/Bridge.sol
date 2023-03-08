// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.4;

import "@openzeppelin/contracts-upgradeable/proxy/utils/Initializable.sol";
import "@openzeppelin/contracts-upgradeable/utils/AddressUpgradeable.sol";
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
    ) external payable returns (uint256) {
        return _enqueueDelayedMessage(kind, sender, messageDataHash, msg.value);
    }

    function _transferFunds(address, uint256) internal override {
        // do nothing as Eth transfer is part of TX execution
    }

    function _executeLowLevelCall(
        address to,
        uint256 value,
        bytes memory data
    ) internal override returns (bool success, bytes memory returnData) {
        // solhint-disable-next-line avoid-low-level-calls
        (success, returnData) = to.call{value: value}(data);
    }

    function _baseFeeToReport() internal override returns (uint256) {
        return block.basefee;
    }
}
