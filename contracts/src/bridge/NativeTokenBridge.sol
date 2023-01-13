// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.4;

import "./Bridge.sol";
import "../libraries/AddressAliasHelper.sol";
import "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import "@openzeppelin/contracts/token/ERC20/utils/SafeERC20.sol";

/// @dev Provided zero address token
error InvalidToken();

/// @dev Provided insufficient value for token fees
error InvalidTokenFeeAmount(uint256);

/// @dev Function not applicable for native token bridge
error NotApplicable();

/**
 * @title Staging ground for incoming and outgoing messages
 * @notice Holds the inbox accumulator for sequenced and delayed messages.
 * Unlike the standard bridge, native token bridge escrows the native token that is used to pay for fees.
 * Since the escrow is held here, this contract also contains a list of allowed
 * outboxes that can make calls from here and withdraw this escrow.
 */
contract NativeTokenBridge is Bridge {
    using SafeERC20 for IERC20;

    address public nativeToken;

    function initialize(IOwnable rollup_, address nativeToken_) external initializer onlyDelegated {
        if (nativeToken_ == address(0)) revert InvalidToken();
        nativeToken = nativeToken_;
        _activeOutbox = EMPTY_ACTIVEOUTBOX;
        rollup = rollup_;
    }

    /// @inheritdoc IBridge
    function enqueueDelayedMessage(
        uint8,
        address,
        bytes32
    ) external payable override returns (uint256) {
        revert NotApplicable();
    }

    function enqueueDelayedMessage(
        uint8 kind,
        address sender,
        bytes32 messageDataHash,
        uint256 tokenFeeAmount
    ) external returns (uint256) {
        if (!this.allowedDelayedInboxes(msg.sender)) revert NotDelayedInbox(msg.sender);

        uint256 messageCount = addMessageToDelayedAccumulator(
            kind,
            sender,
            uint64(block.number),
            uint64(block.timestamp), // solhint-disable-line not-rely-on-time
            block.basefee,
            messageDataHash
        );

        // inbox applies alias to sender, undo it to fetch tokens
        address undoAliasSender = AddressAliasHelper.undoL1ToL2Alias(sender);

        // escrow fee token
        IERC20(nativeToken).safeTransferFrom(undoAliasSender, address(this), tokenFeeAmount);

        return messageCount;
    }
}
