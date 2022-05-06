// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.0;

import "../bridge/IBridge.sol";

interface IRollupEventBridge {
    function bridge() external view returns (IBridge);

    function initialize(address _bridge, address _rollup) external;

    function rollup() external view returns (address);

    function rollupInitialized(uint256 chainId) external;
}
