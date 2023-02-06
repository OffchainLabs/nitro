// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE
// SPDX-License-Identifier: BUSL-1.1

// solhint-disable-next-line compiler-version
pragma solidity >=0.6.9 <0.9.0;

import "./IBridgeCreator.sol";
import "./RollupProxy.sol";
import "../osp/IOneStepProofEntry.sol";
import "../challenge/IChallengeManager.sol";

interface IRollupCreator {
    function setTemplates(
        IBridgeCreator _bridgeCreator,
        IOneStepProofEntry _osp,
        IChallengeManager _challengeManagerLogic,
        IRollupAdmin _rollupAdminLogic,
        IRollupUser _rollupUserLogic,
        address _validatorUtils,
        address _validatorWalletCreator
    ) external;
}

interface IEthRollupCreator is IRollupCreator {
    function createRollup(Config memory config, address expectedRollupAddr)
        external
        returns (address);
}

interface IERC20RollupCreator is IRollupCreator {
    function createRollup(
        Config memory config,
        address expectedRollupAddr,
        address nativeToken
    ) external returns (address);
}
