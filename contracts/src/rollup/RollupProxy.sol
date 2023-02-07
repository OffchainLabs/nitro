// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.0;

import "../libraries/AdminFallbackProxy.sol";
import "./IRollupLogic.sol";

contract RollupProxy is AdminFallbackProxy {
    constructor(Config memory config, ContractDependencies memory connectedContracts)
        AdminFallbackProxy(
            address(connectedContracts.rollupAdminLogic),
            abi.encodeWithSelector(IRollupAdmin.initialize.selector, config, connectedContracts),
            address(connectedContracts.rollupUserLogic),
            abi.encodeWithSelector(IRollupUserAbs.initialize.selector, config.stakeToken),
            config.owner
        )
    {}
}
