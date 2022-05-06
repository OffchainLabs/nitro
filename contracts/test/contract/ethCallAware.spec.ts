/*
 * Copyright 2019-2020, Offchain Labs, Inc.
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

/* eslint-env node, mocha */

import { assert, expect } from "chai";
import { ethers } from "hardhat";
import { EthCallAwareTester__factory } from "../../build/types";
import { TxSuccessEvent } from "../../build/types/src/test-helpers/EthCallAwareTester";
import { initializeAccounts } from "./utils";
import { CallOverrides, Signer } from "ethers";

describe("EthCallAware", async () => {
  let accounts: Signer[];

  const setupEthCallAware = async () => {
    accounts = await initializeAccounts();

    const ethCallAwareFac = (await ethers.getContractFactory(
      "EthCallAwareTester"
    )) as EthCallAwareTester__factory;

    const ethCallAware = await ethCallAwareFac.deploy();
    return ethCallAware.connect(ethers.provider);
  };
  const num = 10;
  const data = "0x2020";

  it(`estimate gas returns correct value`, async () => {
    const ethCallAware = await setupEthCallAware();
    const gasEstimate = await ethCallAware.connect(accounts[0]).estimateGas.testFunction(num, data);

    const res = await ethCallAware.connect(accounts[0]).functions.testFunction(num, data);
    const receipt = await res.wait();
    const event = ethCallAware.interface.parseLog(receipt.logs[0]).args as TxSuccessEvent["args"];
    expect(event.data, "data").to.eq(data);
    expect(event.num.toNumber(), "num").to.eq(num);
    expect(gasEstimate.toNumber(), "gas used").to.eq(receipt.gasUsed);
  });

  it(`doesn't revert if not opt-in eth call`, async () => {
    const ethCallAware = await setupEthCallAware();
    await ethCallAware.callStatic.testFunction(num, data);
  });

  const runTest = async (opts?: CallOverrides) => {
      it(`allows transaction to continue`, async () => {
        const ethCallAware = await setupEthCallAware();
        const res = await ethCallAware.connect(accounts[0]).functions.testFunction(num, data);
        const receipt = await res.wait();

        const event = ethCallAware.interface.parseLog(receipt.logs[0])
          .args as TxSuccessEvent["args"];
        expect(event.data, "data").to.eq(data);
        expect(event.num.toNumber(), "num").to.eq(num);
      });

      it(`call reverts with data`, async () => {
        const ethCallAware = await setupEthCallAware();

        // waiting for release that includes https://github.com/TrueFiEng/Waffle/pull/719
        // await expect(
        //   ethCallAware.callStatic.testFunction(num, data, opts),
        //   "Error message"
        // ).to.be.revertedWith(`CallAwareData(0, "${data}")`);

        const res = await ethCallAware.callStatic
          .testFunction(num, data, opts)
          .catch((err) => err.error.toString());

        const regexp = new RegExp(
          "VM Exception while processing transaction: reverted with custom error '(.*)'"
        );
        const matches = regexp.exec(res);
        assert(matches && matches.length >= 1, "error not found");

        const customErrorThrown = matches[1];
        expect(customErrorThrown).to.equal(`CallAwareData(0, "${data}")`, "wrong error thrown");
      });
  };

  describe('running with overloaded gas price', async () => {
    await runTest({ gasPrice: "0x4404cA11" });
    await runTest({ gasPrice: "0x1234404cA11" });
  })
  describe('running with overloaded origin', async () => {
    await runTest({ from: "0x0000000000000000000000000000000e4404cA11" });
  })
});
