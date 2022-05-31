import { expect } from "chai";
import { ethers, waffle } from "hardhat";
import {
  ValidatorWalletCreator,
  ValidatorWalletCreator__factory,
  ValidatorWallet,
  RollupMock,
} from "../../build/types";
import { initializeAccounts } from "./utils";

type ArrayElement<A> = A extends readonly (infer T)[] ? T : never;

describe("Validator Wallet", () => {
  let accounts: Awaited<ReturnType<typeof initializeAccounts>>;
  let owner: ArrayElement<typeof accounts>;
  let executor: ArrayElement<typeof accounts>;
  let walletCreator: ValidatorWalletCreator;
  let wallet: ValidatorWallet;
  let rollupMock: RollupMock;

  before(async () => {
    accounts = await initializeAccounts();
    const WalletCreator: ValidatorWalletCreator__factory = await ethers.getContractFactory(
      "ValidatorWalletCreator"
    );
    walletCreator = await WalletCreator.deploy();
    await walletCreator.deployed();

    owner = await accounts[0];
    executor = await accounts[1];
    const walletCreationTx = await (await walletCreator.createWallet()).wait();

    const events = walletCreationTx.logs
      .filter((curr) => curr.topics[0] === walletCreator.interface.getEventTopic("WalletCreated"))
      .map((curr) => walletCreator.interface.parseLog(curr));
    if (events.length !== 1) throw new Error("No Events!");
    const walletAddr = events[0].args.walletAddress;
    const Wallet = await ethers.getContractFactory("ValidatorWallet");
    wallet = (await Wallet.attach(walletAddr)) as ValidatorWallet;

    await wallet.setExecutor([await executor.getAddress()], [true])
    await wallet.transferOwnership(await owner.getAddress())

    const RollupMock = await ethers.getContractFactory("RollupMock");
    rollupMock = (await RollupMock.deploy()) as RollupMock;

    await accounts[0].sendTransaction({
      to: wallet.address,
      value: ethers.utils.parseEther("5"),
    });
  });

  it("should validate function signatures", async function () {
    const funcSigs = ["0x12345678", "0x00000001", "0x01230000"];
    await wallet.setOnlyOwnerFunctionSigs(funcSigs, [true, true, true]);

    expect(await wallet.onlyOwnerFuncSigs("0x01230000")).to.be.true;
    expect(await wallet.onlyOwnerFuncSigs("0x00000001")).to.be.true;
    expect(await wallet.onlyOwnerFuncSigs("0x12345678")).to.be.true;

    // should succeed if random signature
    await wallet.connect(executor).validateExecuteTransaction("0x987698769876");

    for (const sig of funcSigs) {
      // owner should succeed
      await expect(wallet.connect(owner).validateExecuteTransaction(sig));
      // executor should revert
      await expect(wallet.connect(executor).validateExecuteTransaction(sig)).to.be.reverted;
      // TODO: update waffle once released with fix for custom errors https://github.com/TrueFiEng/Waffle/pull/719
    }
  });

  it("should not allow executor to execute certain txs", async function () {
    const data = rollupMock.interface.encodeFunctionData("withdrawStakerFunds", [
      await executor.getAddress(),
    ]);

    const expectedSig = "0x81fbc98a";
    expect(data.substring(0, 10)).to.equal(expectedSig);
    expect(await wallet.onlyOwnerFuncSigs(expectedSig)).to.be.true;

    await expect(
      wallet.connect(executor).executeTransaction(data, rollupMock.address, 0)
    ).to.be.revertedWith(
      `OnlyOwnerFunctionSig("${await owner.getAddress()}", "${await executor.getAddress()}", "${expectedSig}")`
    );
    await expect(wallet.connect(owner).executeTransaction(data, rollupMock.address, 0)).to.emit(
      rollupMock,
      "WithdrawTriggered"
    );
  });

  it("should allow regular functions to be executed", async function () {
    const data = rollupMock.interface.encodeFunctionData("removeOldZombies", [0]);

    const expectedSig = "0xedfd03ed";
    expect(data.substring(0, 10)).to.equal(expectedSig);
    expect(await wallet.onlyOwnerFuncSigs(expectedSig)).to.be.false;

    await expect(wallet.connect(executor).executeTransaction(data, rollupMock.address, 0)).to.emit(
      rollupMock,
      "ZombieTriggered"
    );
    await expect(wallet.connect(owner).executeTransaction(data, rollupMock.address, 0)).to.emit(
      rollupMock,
      "ZombieTriggered"
    );
  });

  it("should handle fallback function", async function () {
    const dest = await accounts[3].getAddress();
    const amount = ethers.utils.parseEther("0.2");

    const expectedSig = "0x00000000"
    await wallet.setOnlyOwnerFunctionSigs([expectedSig], [true]);

    await expect(
      wallet.connect(executor).executeTransaction("0x", dest, amount)
    ).to.be.revertedWith(
      `OnlyOwnerFunctionSig("${await owner.getAddress()}", "${await executor.getAddress()}", "${expectedSig}")`
    );

    await wallet.setOnlyOwnerFunctionSigs([expectedSig], [false]);

    const prevBal = await waffle.provider.getBalance(dest);
    await wallet.connect(executor).executeTransaction("0x", dest, amount);
    const postBal = await waffle.provider.getBalance(dest);

    expect(prevBal.add(amount)).to.equal(postBal);
  });

  it("should reject batch if single tx is not allowed by executor", async function () {
    const data = [
      rollupMock.interface.encodeFunctionData("removeOldZombies", [0]),
      rollupMock.interface.encodeFunctionData("withdrawStakerFunds", [await executor.getAddress()]),
    ];

    const expectedSig = data[1].substring(0, 10)

    await expect(
      wallet
        .connect(executor)
        .executeTransactions(data, [rollupMock.address, rollupMock.address], [0, 0])
    ).to.be.revertedWith(
      `OnlyOwnerFunctionSig("${await owner.getAddress()}", "${await executor.getAddress()}", "${expectedSig}")`
    );
  });
});
