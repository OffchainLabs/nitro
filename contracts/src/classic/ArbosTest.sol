// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity >=0.4.21 <0.9.0;

interface ArbosTest {
    function installAccount(
        address addr,
        bool isEOA,
        uint256 balance,
        uint256 nonce,
        bytes calldata code,
        bytes calldata initStorage
    ) external;

    function getMarshalledStorage(address addr) external view; // returns raw returndata

    function getAccountInfo(address addr) external view; // returns raw returndata

    function burnArbGas(uint256 gasAmount) external view;

    function setNonce(address addr, uint256 nonce) external;

    function setBalance(address addr, uint256 balance) external;

    function setCode(address addr, bytes calldata code) external;

    function setState(address addr, bytes calldata state) external;

    function store(
        address addr,
        uint256 key,
        uint256 value
    ) external;

    function getAllAccountAddresses() external view returns (bytes memory);

    function getAllAccountHashes() external view returns (bytes memory);

    function getSerializedEVMState(address addr) external view returns (bytes memory);
}
