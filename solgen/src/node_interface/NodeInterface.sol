// SPDX-License-Identifier: Apache-2.0

pragma solidity >=0.4.21 <0.9.0;

/** @title Interface for providing gas estimation for retryable auto-redeems and constructing outbox proofs
 *  @notice This contract doesn't exist on-chain. Instead it is a virtual interface accessible at
 *  0x00000000000000000000000000000000000000C8
 *  This is a cute trick to allow an Arbitrum node to provide data without us having to implement additional RPCs
 */

interface NodeInterface {
    /**
     * @notice Estimate the cost of putting a message in the L2 inbox that is reexecuted
     * @param sender sender of the L1 and L2 transaction
     * @param deposit amount to deposit to sender in L2
     * @param to destination L2 contract address
     * @param l2CallValue call value for retryable L2 message
     * @param excessFeeRefundAddress gasLimit x maxFeePerGas - execution cost gets credited here on L2 balance
     * @param callValueRefundAddress l2Callvalue gets credited here on L2 if retryable txn times out or gets cancelled
     * @param data ABI encoded data of L2 message
     */
    function estimateRetryableTicket(
        address sender,
        uint256 deposit,
        address to,
        uint256 l2CallValue,
        address excessFeeRefundAddress,
        address callValueRefundAddress,
        bytes calldata data
    ) external;

    /**
     * @notice Constructs an outbox proof of an l2->l1 send's existence in the outbox accumulator
     * @param size the number of elements in the accumulator
     * @param leaf the position of the send in the accumulator
     * @return send the l2->l1 send's hash
     * @return root the root of the outbox accumulator
     * @return proof level-by-level branch hashes constituting a proof of the send's membership at the given size
     */
    function constructOutboxProof(uint64 size, uint64 leaf)
        external
        view
        returns (
            bytes32 send,
            bytes32 root,
            bytes32[] memory proof
        );
}
