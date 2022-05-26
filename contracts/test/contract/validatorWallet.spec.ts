import { expect } from "chai";
import { assert } from "console";
import { ethers } from "hardhat";
import {
  ValidatorWalletCreator,
  ValidatorWalletCreator__factory,
  ValidatorWallet,
} from "../../build/types";
import { initializeAccounts } from "./utils";

type ArrayElement<A> = A extends readonly (infer T)[] ? T : never;

describe("Validator Wallet", () => {
  let accounts: Awaited<ReturnType<typeof initializeAccounts>>;
  let owner: ArrayElement<typeof accounts>;
  let executor: ArrayElement<typeof accounts>;
  let walletCreator: ValidatorWalletCreator;
  let wallet: ValidatorWallet;

  before(async () => {
    accounts = await initializeAccounts();
    const WalletCreator: ValidatorWalletCreator__factory = await ethers.getContractFactory(
      "ValidatorWalletCreator"
    );
    walletCreator = await WalletCreator.deploy();
    await walletCreator.deployed();

    owner = await accounts[0];
    executor = await accounts[1];
    const walletCreationTx = await (
      await walletCreator.createWallet(await executor.getAddress(), await owner.getAddress())
    ).wait();

    const events = walletCreationTx.logs
      .filter((curr) => curr.topics[0] === walletCreator.interface.getEventTopic("WalletCreated"))
      .map((curr) => walletCreator.interface.parseLog(curr));
    if (events.length !== 1) throw new Error("No Events!");
    const walletAddr = events[0].args.walletAddress;
    const Wallet = await ethers.getContractFactory("ValidatorWallet");
    wallet = (await Wallet.attach(walletAddr)) as ValidatorWallet;
  });

  it("should validate function signatures", async function () {
    const funcSigs = ["0x12345678", "0x00000000", "0x01230000"];
    await wallet.setOnlyOwnerFunctionSigs(funcSigs, [true, true, true]);

    expect(await wallet.onlyOwnerFuncSigs("0x01230000")).to.be.true;
    expect(await wallet.onlyOwnerFuncSigs("0x00000000")).to.be.true;
    expect(await wallet.onlyOwnerFuncSigs("0x12345678")).to.be.true;

    // should succeed if random signature
    await wallet.connect(owner).validateExecuteTransaction("0x987698769876");

    for (const sig of funcSigs) {
      await expect(wallet.connect(owner).validateExecuteTransaction(sig)).to.be.reverted;
      // TODO: update waffle once released with fix for custom errors https://github.com/TrueFiEng/Waffle/pull/719
    }
  });

  // TODO: example calling random func with params and failing
  // fallback function example
});
