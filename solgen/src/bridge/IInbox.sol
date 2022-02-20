//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
// SPDX-License-Identifier: UNLICENSED
//

pragma solidity ^0.8.4;

import "./IBridge.sol";
import "./IMessageProvider.sol";
import "../libraries/Error.sol";

/// @dev The contract is paused, so cannot be paused
error AlreadyPaused();

/// @dev The contract is unpaused, so cannot be unpaused
error AlreadyUnpaused();

/// @dev The contract is paused
error Paused();

interface IInbox is IMessageProvider {
    function sendL2Message(bytes calldata messageData) external returns (uint256);

    function sendUnsignedTransaction(
        uint256 gasLimit,
        uint256 gasFeeCap,
        uint256 nonce,
        address to,
        uint256 amount,
        bytes calldata data
    ) external returns (uint256);

    function sendContractTransaction(
        uint256 gasLimit,
        uint256 gasFeeCap,
        address to,
        uint256 amount,
        bytes calldata data
    ) external returns (uint256);

    function sendL1FundedUnsignedTransaction(
        uint256 gasLimit,
        uint256 gasFeeCap,
        uint256 nonce,
        address to,
        bytes calldata data
    ) external payable returns (uint256);

    function sendL1FundedContractTransaction(
        uint256 gasLimit,
        uint256 gasFeeCap,
        address to,
        bytes calldata data
    ) external payable returns (uint256);

    function createRetryableTicket(
        address to,
        uint256 arbTxCallValue,
        uint256 maxSubmissionCost,
        address submissionRefundAddress,
        address valueRefundAddress,
        uint256 gasLimit,
        uint256 gasFeeCap,
        bytes calldata data
    ) external payable returns (uint256);

    function depositEth(uint256 maxSubmissionCost) external payable returns (uint256);

    function bridge() external view returns (IBridge);
}
