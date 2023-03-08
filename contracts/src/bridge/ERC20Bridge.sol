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

/**
 * @title Staging ground for incoming and outgoing messages
 * @notice Unlike the standard Eth bridge, native token bridge escrows the custom ERC20 token which is
 * used as native currency on L2.
 */
contract ERC20Bridge is AbsBridge, IERC20Bridge {
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
        return _enqueueDelayedMessage(kind, sender, messageDataHash, tokenFeeAmount);
    }

    function _transferFunds(address sender, uint256 amount) internal override {
        // inbox applies alias to sender, undo it to fetch tokens
        address undoAliasSender = AddressAliasHelper.undoL1ToL2Alias(sender);

        // escrow fee token
        IERC20(nativeToken).safeTransferFrom(undoAliasSender, address(this), amount);
    }

    function _executeLowLevelCall(
        address to,
        uint256 value,
        bytes memory data
    ) internal override returns (bool success, bytes memory returnData) {
        // first release native token
        IERC20(nativeToken).safeTransfer(to, value);
        success = true;

        // if there's data do additional contract call
        if (data.length > 0) {
            // solhint-disable-next-line avoid-low-level-calls
            (success, returnData) = to.call(data);
        }
    }
}
