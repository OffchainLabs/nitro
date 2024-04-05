// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro-contracts/blob/main/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity >=0.4.21 <0.9.0;

/** @title An extension to NodeInterface not meant for public consumption. Do not call.
 *  @notice This contract doesn't exist on-chain. Instead it is a virtual interface accessible at 0xc9.
 *  These methods add additional debugging and network monitoring instruments not intended for end users and
 *  as such may change without notice.
 */

interface NodeInterfaceDebug {
    struct RetryableInfo {
        uint64 timeout;
        address from;
        address to;
        uint256 value;
        address beneficiary;
        uint64 tries;
        bytes data;
    }

    /**
     * @notice gets a retryable
     * @param ticket the retryable's id
     * @return retryable the serialized retryable
     */
    function getRetryable(bytes32 ticket) external view returns (RetryableInfo memory retryable);
}
