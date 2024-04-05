// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro-contracts/blob/main/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity >=0.4.21 <0.9.0;

/**
 * @title Methods for managing retryables.
 * @notice Precompiled contract in every Arbitrum chain for retryable transaction related data retrieval and interactions. Exists at 0x000000000000000000000000000000000000006e
 */
interface ArbRetryableTx {
    /**
     * @notice Schedule an attempt to redeem a redeemable tx, donating all of the call's gas to the redeem.
     * Revert if ticketId does not exist.
     * @param ticketId unique identifier of retryable message: keccak256(keccak256(ArbchainId, inbox-sequence-number), uint(0) )
     * @return txId that the redeem attempt will have
     */
    function redeem(bytes32 ticketId) external returns (bytes32);

    /**
     * @notice Return the minimum lifetime of redeemable txn.
     * @return lifetime in seconds
     */
    function getLifetime() external view returns (uint256);

    /**
     * @notice Return the timestamp when ticketId will age out, reverting if it does not exist
     * @param ticketId unique ticket identifier
     * @return timestamp for ticket's deadline
     */
    function getTimeout(bytes32 ticketId) external view returns (uint256);

    /**
     * @notice Adds one lifetime period to the life of ticketId.
     * Donate gas to pay for the lifetime extension.
     * If successful, emits LifetimeExtended event.
     * Revert if ticketId does not exist, or if the timeout of ticketId is already at least one lifetime period in the future.
     * @param ticketId unique ticket identifier
     * @return new timeout of ticketId
     */
    function keepalive(bytes32 ticketId) external returns (uint256);

    /**
     * @notice Return the beneficiary of ticketId.
     * Revert if ticketId doesn't exist.
     * @param ticketId unique ticket identifier
     * @return address of beneficiary for ticket
     */
    function getBeneficiary(bytes32 ticketId) external view returns (address);

    /**
     * @notice Cancel ticketId and refund its callvalue to its beneficiary.
     * Revert if ticketId doesn't exist, or if called by anyone other than ticketId's beneficiary.
     * @param ticketId unique ticket identifier
     */
    function cancel(bytes32 ticketId) external;

    /**
     * @notice Gets the redeemer of the current retryable redeem attempt.
     * Returns the zero address if the current transaction is not a retryable redeem attempt.
     * If this is an auto-redeem, returns the fee refund address of the retryable.
     */
    function getCurrentRedeemer() external view returns (address);

    /**
     * @notice Do not call. This method represents a retryable submission to aid explorers.
     * Calling it will always revert.
     */
    function submitRetryable(
        bytes32 requestId,
        uint256 l1BaseFee,
        uint256 deposit,
        uint256 callvalue,
        uint256 gasFeeCap,
        uint64 gasLimit,
        uint256 maxSubmissionFee,
        address feeRefundAddress,
        address beneficiary,
        address retryTo,
        bytes calldata retryData
    ) external;

    event TicketCreated(bytes32 indexed ticketId);
    event LifetimeExtended(bytes32 indexed ticketId, uint256 newTimeout);
    event RedeemScheduled(
        bytes32 indexed ticketId,
        bytes32 indexed retryTxHash,
        uint64 indexed sequenceNum,
        uint64 donatedGas,
        address gasDonor,
        uint256 maxRefund,
        uint256 submissionFeeRefund
    );
    event Canceled(bytes32 indexed ticketId);

    /// @dev DEPRECATED in favour of new RedeemScheduled event after the nitro upgrade
    event Redeemed(bytes32 indexed userTxHash);

    error NoTicketWithID();
    error NotCallable();
}
