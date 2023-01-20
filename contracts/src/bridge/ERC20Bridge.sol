// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.4;

import "./AbsBridge.sol";
import "./IERC20Bridge.sol";
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
 * @notice Unlike the standard Eth bridge, native token bridge escrows the custom ERC20 token which is
 * used as native currency on L2.
 */
contract ERC20Bridge is AbsBridge, IERC20Bridge {
    using AddressUpgradeable for address;
    using SafeERC20 for IERC20;

    /// @inheritdoc IERC20Bridge
    address public nativeToken;

    /// @inheritdoc IERC20Bridge
    function initialize(IOwnable rollup_, address nativeToken_) external initializer onlyDelegated {
        if (nativeToken_ == address(0)) revert InvalidToken();
        nativeToken = nativeToken_;
        _activeOutbox = EMPTY_ACTIVEOUTBOX;
        rollup = rollup_;
    }

    /// @inheritdoc IERC20Bridge
    function enqueueDelayedMessage(
        uint8 kind,
        address sender,
        bytes32 messageDataHash,
        uint256 tokenFeeAmount
    ) external returns (uint256) {
        if (!allowedDelayedInboxes(msg.sender)) revert NotDelayedInbox(msg.sender);

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

        // first release native token
        // solhint-disable-next-line avoid-low-level-calls
        (success, returnData) = nativeToken.call(
            abi.encodeWithSelector(IERC20.transfer.selector, to, value)
        );

        // if there's data do additional contract call (if token transfer was succesful)
        if (data.length > 0) {
            if (success) {
                // solhint-disable-next-line avoid-low-level-calls
                (success, returnData) = to.call(data);
            }
        }

        _activeOutbox = prevOutbox;
        emit BridgeCallTriggered(msg.sender, to, value, data);
    }
}
