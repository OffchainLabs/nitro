import { ethers, getNamedAccounts } from "hardhat"
import { expect } from "chai";


describe("Admin Aware Proxy test", function () {
    it("Should deploy proxy correctly", async function () {
        const { deployer } = await getNamedAccounts()
        const admin = "0x1000000000000000000000000000000000000001"

        const Proxy = await ethers.getContractFactory("TransparentUpgradeableProxy")
        const AdminAwareProxy = await ethers.getContractFactory("AdminAwareProxy")
        const ProxyTesterLogic = await ethers.getContractFactory("ProxyTesterLogic")

        const logicA = await ProxyTesterLogic.deploy()
        await logicA.deployTransaction.wait()

        const logicB = await ProxyTesterLogic.deploy()
        await logicB.deployTransaction.wait()

        const adminAwareProxy = await AdminAwareProxy.deploy()
        await adminAwareProxy.deployTransaction.wait()

        const proxyTemp =  await Proxy.deploy(adminAwareProxy.address, admin, "0x")
        await proxyTemp.deployTransaction.wait()

        const proxyAddr = proxyTemp.address;
        const proxy = AdminAwareProxy.attach(proxyAddr)

        const initParams = [
          {
            confirmPeriodBlocks: 2,
            extraChallengeTimeBlocks: 1,
            stakeToken: ethers.constants.AddressZero,
            baseStake: 1,
            wasmModuleRoot: ethers.constants.HashZero,
            owner: admin,
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
        const init = await proxy.initialize(...initParams);
        await init.wait()

        const proxyLogic = ProxyTesterLogic.attach(proxyAddr)
        const prevOwner = await proxy.owner()
        expect(admin).to.equal(prevOwner)

        const expectedNewOwner = "0x0000000001023012301203120301000000000102";
        const setOwner = await proxyLogic.setOwner(expectedNewOwner)
        await setOwner.wait()

        const newOwner = await proxy.owner()
        expect(expectedNewOwner).to.equal(newOwner)
    });
})
