// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.0;

import "@openzeppelin/contracts/proxy/transparent/ProxyAdmin.sol";
import "@openzeppelin/contracts/proxy/transparent/TransparentUpgradeableProxy.sol";
import "@openzeppelin/contracts/access/Ownable.sol";

import "./ValidatorWallet.sol";

contract ValidatorWalletCreator is Ownable {
    event WalletCreated(
        address indexed walletAddress,
        address indexed userAddress,
        address adminProxy
    );
    event TemplateUpdated();

    address public template;

    constructor() Ownable() {
        template = address(new ValidatorWallet());
    }

    function setTemplate(address _template) external onlyOwner {
        template = _template;
        emit TemplateUpdated();
    }

    function createWallet() external returns (address) {
        ProxyAdmin admin = new ProxyAdmin();
        address proxy = address(
            new TransparentUpgradeableProxy(address(template), address(admin), "")
        );
        admin.transferOwnership(msg.sender);
        ValidatorWallet(proxy).initialize();
        ValidatorWallet(proxy).transferOwnership(msg.sender);
        emit WalletCreated(proxy, msg.sender, address(admin));
        return proxy;
    }
}
