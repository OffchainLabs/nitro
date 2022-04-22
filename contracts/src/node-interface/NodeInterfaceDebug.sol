// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity >=0.4.21 <0.9.0;

/** @title An extension to NodeInterface not meant for public consumption. Do not call.
 *  @notice This contract doesn't exist on-chain. Instead it is a virtual interface accessible at 0xc9.
 *  These methods add additional debugging and network monitoring instruments not intended for end users and
 *  as such may change without notice.
 */

interface NodeInterfaceDebug {
    /**
     * @notice exports the state of the retryable timeout queue
     * @return queueSize the number of elements in the queue
     * @return tickets the ordered entries of the queue
     * @return timeouts the timeout associated with each element in the queue
     */
    function retryableTimeoutQueue()
        external
        view
        returns (
            uint64 queueSize,
            bytes32[] memory tickets,
            uint64[] memory timeouts
        );

    /**
     * @notice serializes a retryable
     * @return retryable the serialized retryable
     */
    function serializeRetryable(bytes32 ticket) external view returns (bytes memory retryable);
}
