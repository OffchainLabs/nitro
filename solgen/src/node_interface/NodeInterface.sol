// SPDX-License-Identifier: Apache-2.0

pragma solidity >=0.4.21 <0.9.0;

/** @title Interface for providing gas estimation for retryable auto-redeems
 *  @notice This contract doesn't exist on-chain. Instead it is a virtual interface accessible at 0x00000000000000000000000000000000000000C8
 *  This is a cute trick to allow an Arbitrum node to provide data without us having to implement an additional RPC
 */

interface NodeInterface {
    /**
     * @notice Estimate the cost of putting a message in the L2 inbox that is reexecuted
     * @param sender sender of the L1 and L2 transaction
     * @param deposit amount to deposit to sender in L2
     * @param destAddr destination L2 contract address
     * @param l2CallValue call value for retryable L2 message
     * @param excessFeeRefundAddress maxgas x gasprice - execution cost gets credited here on L2 balance
     * @param callValueRefundAddress l2Callvalue gets credited here on L2 if retryable txn times out or gets cancelled
     * @param data ABI encoded data of L2 message
     */
    function estimateRetryableTicket(
        address sender,
        uint256 deposit,
        address destAddr,
        uint256 l2CallValue,
        address excessFeeRefundAddress,
        address callValueRefundAddress,
        bytes calldata data
    ) external;
}
