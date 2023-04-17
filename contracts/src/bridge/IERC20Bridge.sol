// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE
// SPDX-License-Identifier: BUSL-1.1

// solhint-disable-next-line compiler-version
pragma solidity >=0.6.9 <0.9.0;

import "./IOwnable.sol";
import "./IBridge.sol";

interface IERC20Bridge is IBridge {
    /**
     * @dev token that is escrowed in bridge on L1 side and minted on L2 as native currency.
     * Also fees are paid in this token. ERC777, fee on transfer tokens and rebasing tokens
     * are not supported to be used as chain's native token, as they can break collateralization
     * invariants.
     */
    function nativeToken() external returns (address);

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
