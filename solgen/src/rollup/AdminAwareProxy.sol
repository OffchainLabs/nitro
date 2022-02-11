// SPDX-License-Identifier: Apache-2.0

/*
 * Copyright 2021, Offchain Labs, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

pragma solidity ^0.8.0;

import "@openzeppelin/contracts/proxy/Proxy.sol";
import "@openzeppelin/contracts/utils/Address.sol";
import "@openzeppelin/contracts/utils/StorageSlot.sol";

import "./RollupCore.sol";
import "./RollupEventBridge.sol";
import "./RollupLib.sol";
import "./Node.sol";
import "./IRollupLogic.sol";

import "../libraries/Cloneable.sol";
import "../libraries/ProxyUtil.sol";

struct ContractDependencies {
    IBridge delayedBridge;
    ISequencerInbox sequencerInbox;
    IOutbox outbox;
    RollupEventBridge rollupEventBridge;
    IBlockChallengeFactory blockChallengeFactory;

    IRollupAdmin rollupAdminLogic;
    IRollupUser rollupUserLogic;
}

library AAPLib {
    using Address for address;

    // bytes32(uint256(keccak256("arbitrum.aap.owner")) - 1)
    bytes32 internal constant AAP_OWNER_SLOT = 0x6bc411416ceafb20f5c538ed5d690d0e7c2cfe9edcad838ef83bd48f7078c477;
    
    // bytes32(uint256(keccak256("arbitrum.aap.logic.admin")) - 1)
    bytes32 internal constant AAP_ADMIN_LOGIC_SLOT = 0x06f3f672ac970b8d9bde2a8a9d25e98aea70edda39c801715dcc26271150c253;
    
    // bytes32(uint256(keccak256("arbitrum.aap.logic.user")) - 1)
    bytes32 internal constant AAP_USER_LOGIC_SLOT = 0x823928b9666b737108900c1fff17aa4166fe3fbb486b1f1dcfc1ba46e592e512;
    

    // based on OZ proxies
    // https://github.com/OpenZeppelin/openzeppelin-contracts/blob/5b6112000c2e1b61db63d7b0bb33ab0775ec0975/contracts/proxy/ERC1967/ERC1967Upgrade.sol#L42-L48
    function setAAPOwner(address newOwner) internal {
        StorageSlot.getAddressSlot(AAP_OWNER_SLOT).value = newOwner;
    }

    function setAAPAdminLogic(address newAdminLogic) internal {
        StorageSlot.getAddressSlot(AAP_ADMIN_LOGIC_SLOT).value = newAdminLogic;
    }

    function setAAPUserLogic(address newUserLogic) internal {
        StorageSlot.getAddressSlot(AAP_USER_LOGIC_SLOT).value = newUserLogic;
    }

    function getAAPOwner() internal view returns (address) {
        return StorageSlot.getAddressSlot(AAP_OWNER_SLOT).value;
    }

    function getAAPAdminLogic() internal view returns (address) {
        return StorageSlot.getAddressSlot(AAP_ADMIN_LOGIC_SLOT).value;
    }

    function getAAPUserLogic() internal view returns (address) {
        return StorageSlot.getAddressSlot(AAP_USER_LOGIC_SLOT).value;
    }
}


contract AdminAwareProxy is Cloneable, Proxy {
    using Address for address;

    function owner() public view returns (address) {
        return AAPLib.getAAPOwner();
    }

    function adminLogic() public view returns (address) {
        return AAPLib.getAAPAdminLogic();
    }

    function userLogic() public view returns (address) {
        return AAPLib.getAAPUserLogic();
    }

    // _rollupParams = [ confirmPeriodBlocks, extraChallengeTimeBlocks, chainId, baseStake ]
    // connectedContracts = [delayedBridge, sequencerInbox, outbox, rollupEventBridge, blockChallengeFactory]
    // sequencerInboxParams = [ maxDelayBlocks, maxFutureBlocks, maxDelaySeconds, maxFutureSeconds ]
    function initialize(
        RollupLib.Config calldata config,
        ContractDependencies calldata connectedContracts
    ) external {
        // AAP expects to be deployed behind a TransparentUpgradeableProxy
        require(!isMasterCopy, "NO_INIT_MASTER");
        // check that correct slots are set in AAPLib
        // hashes aren't calculated during compile time, so we hardcode the value and verify once during init
        require(AAPLib.AAP_OWNER_SLOT == bytes32(uint256(keccak256("arbitrum.aap.owner")) - 1), "WRONG_OWNER_SLOT");
        require(AAPLib.AAP_ADMIN_LOGIC_SLOT == bytes32(uint256(keccak256("arbitrum.aap.logic.admin")) - 1), "WRONG_ADMIN_LOGIC_SLOT");
        require(AAPLib.AAP_USER_LOGIC_SLOT == bytes32(uint256(keccak256("arbitrum.aap.logic.user")) - 1), "WRONG_USER_LOGIC_SLOT");
        
        require(adminLogic() == address(0) && userLogic() == address(0) && owner() == address(0), "ALREADY_INIT");

        require(address(connectedContracts.rollupAdminLogic).isContract(), "ADMIN_LOGIC_NOT_CONTRACT");
        require(address(connectedContracts.rollupUserLogic).isContract(), "USER_LOGIC_NOT_CONTRACT");

        AAPLib.setAAPOwner(config.owner);
        AAPLib.setAAPAdminLogic(address(connectedContracts.rollupAdminLogic));
        AAPLib.setAAPUserLogic(address(connectedContracts.rollupUserLogic));

        (bool successAdmin, ) = address(connectedContracts.rollupAdminLogic).delegatecall(
            abi.encodeWithSelector(AdminAwareProxy.initialize.selector, config, connectedContracts)
        );
        require(successAdmin, "FAIL_INIT_ADMIN_LOGIC");

        (bool successUser, ) = address(connectedContracts.rollupUserLogic).delegatecall(
            abi.encodeWithSelector(AdminAwareProxy.initialize.selector, config, connectedContracts)
        );
        require(successUser, "FAIL_INIT_USER_LOGIC");
    }

    function postUpgradeInit() external {
        // it is assumed the rollup contract is behind a Proxy controlled by a proxy admin
        // this function can only be called by the proxy admin contract
        // this is a ERC1967 proxy admin, which is different to the AAP owner
        address proxyAdmin = ProxyUtil.getProxyAdmin();
        require(msg.sender == proxyAdmin, "NOT_FROM_ADMIN");
    }

    /**
     * @dev This is a virtual function that should be overriden so it returns the address to which the fallback function
     * and {_fallback} should delegate.
     */
    function _implementation()
        internal
        view
        override
        returns (address)
    {
        require(msg.data.length >= 4, "NO_FUNC_SIG");
        address rollupOwner = owner();
        // if there is an owner and it is the sender, delegate to admin logic
        address target = rollupOwner != address(0) && rollupOwner == msg.sender
            ? adminLogic()
            : userLogic();
        require(target.isContract(), "TARGET_NOT_CONTRACT");
        return target;
    }
}
