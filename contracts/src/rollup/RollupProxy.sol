// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.0;

import "../libraries/AdminFallbackProxy.sol";
import "./IRollupAdmin.sol";
import "./Config.sol";

contract RollupProxy is AdminFallbackProxy {
    function initializeProxy(Config memory config, ContractDependencies memory connectedContracts)
        external
    {
        if (
            _getAdmin() == address(0) &&
            _getImplementation() == address(0) &&
            _getSecondaryImplementation() == address(0)
        ) {
            _initialize(
                address(connectedContracts.rollupAdminLogic),
                abi.encodeCall(
                    IRollupAdmin.initialize,
                    (config,
                    connectedContracts)
                ),
                address(connectedContracts.rollupUserLogic),
                abi.encodeCall(IRollupUser.initialize, (config.stakeToken)),
                config.owner
            );
        } else {
            _fallback();
        }
    }
}
