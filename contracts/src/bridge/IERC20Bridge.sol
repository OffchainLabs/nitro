// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro-contracts/blob/main/LICENSE
// SPDX-License-Identifier: BUSL-1.1

// solhint-disable-next-line compiler-version
pragma solidity >=0.6.9 <0.9.0;

import "./IOwnable.sol";
import "./IBridge.sol";

interface IERC20Bridge is IBridge {
    /**
     * @dev token that is escrowed in bridge on L1 side and minted on L2 as native currency.
     * Fees are paid in this token. There are certain restrictions on the native token:
     *  - The token can't be rebasing or have a transfer fee
     *  - The token must only be transferrable via a call to the token address itself
     *  - The token must only be able to set allowance via a call to the token address itself
     *  - The token must not have a callback on transfer, and more generally a user must not be able to make a transfer to themselves revert
     */
    function nativeToken() external view returns (address);

    /**
     * @dev Enqueue a message in the delayed inbox accumulator.
     *      These messages are later sequenced in the SequencerInbox, either
     *      by the sequencer as part of a normal batch, or by force inclusion.
     */
    function enqueueDelayedMessage(
        uint8 kind,
        address sender,
        bytes32 messageDataHash,
        uint256 tokenFeeAmount
    ) external returns (uint256);

    // ---------- initializer ----------

    function initialize(IOwnable rollup_, address nativeToken_) external;
}
