import { ethers } from "hardhat"
import { ProxyTesterLogic, AdminAwareProxy__factory, ProxyTesterLogic__factory, TransparentUpgradeableProxy__factory } from "../typechain-types"
import { ConfigStruct, ContractDependenciesStruct } from "../typechain-types/AdminAwareProxy"
import fs from "fs";
import assert from "assert";
import { expect } from "chai";


describe("Admin Aware Proxy test", function () {
    
    it("Should deploy proxy correctly", async function () {
        const accounts = await ethers.getSigners()
        const Proxy = <TransparentUpgradeableProxy__factory>await ethers.getContractFactory("TransparentUpgradeableProxy")
        const AdminAwareProxy = <AdminAwareProxy__factory>await ethers.getContractFactory("AdminAwareProxy")
        const ProxyTesterLogic = <ProxyTesterLogic__factory>await ethers.getContractFactory("ProxyTesterLogic")

        const logicA = await ProxyTesterLogic.deploy()
        const logicB = await ProxyTesterLogic.deploy()
        const adminAwareProxy = await AdminAwareProxy.deploy()

        const proxyAddr = (
          await Proxy.deploy(adminAwareProxy.address, accounts[1].address, "0x")
        ).address;
        const proxy = AdminAwareProxy.attach(proxyAddr)

        const initParams: [ConfigStruct, ContractDependenciesStruct] = [
          {
            confirmPeriodBlocks: 2,
            extraChallengeTimeBlocks: 1,
            stakeToken: ethers.constants.AddressZero,
            baseStake: 1,
            wasmModuleRoot: ethers.constants.HashZero,
            owner: accounts[1].address,
            chainId: 4216111,
            sequencerInboxMaxTimeVariation: {
              delayBlocks: 1,
              futureBlocks: 1,
              delaySeconds: 1,
              futureSeconds: 1,
            },
          },
          {
            delayedBridge: ethers.constants.AddressZero,
            sequencerInbox: ethers.constants.AddressZero,
            outbox: ethers.constants.AddressZero,
            rollupEventBridge: ethers.constants.AddressZero,
            blockChallengeFactory: ethers.constants.AddressZero,
            rollupAdminLogic: logicA.address,
            rollupUserLogic: logicB.address,
          },
        ];

        await expect(adminAwareProxy.initialize(...initParams)).to.be.revertedWith("NO_INIT_MASTER")
        await proxy.initialize(...initParams);

        const proxyLogic = ProxyTesterLogic.attach(proxyAddr)
        const prevOwner = await proxyLogic.owner()
        const prevOwnerOriginalProxy = await proxy.owner()
        
        expect(accounts[1].address).to.equal(prevOwnerOriginalProxy)
        expect(accounts[1].address).to.equal(prevOwner)
        
        const expectedNewOwner = "0x0000000001023012301203120301000000000102";
        await proxyLogic.setOwner(expectedNewOwner)
        const newOwner = await proxyLogic.owner()
        const newOwnerOriginalProxy = await proxy.owner()

        expect(expectedNewOwner).to.equal(newOwner)
        expect(expectedNewOwner).to.equal(newOwnerOriginalProxy)
    });
})