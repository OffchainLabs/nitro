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
import { TxSuccessEvent } from "../../build/types/EthCallAwareTester";
import { initializeAccounts } from "./utils";

describe("EthCallAware", async () => {
  const setupEthCallAware = async () => {
    const accounts = await initializeAccounts();
    const admin = accounts[0];

    const ethCallAwareFac = (await ethers.getContractFactory(
      "EthCallAwareTester"
    )) as EthCallAwareTester__factory;

    const ethCallAware = (await ethCallAwareFac.deploy()).connect(admin);
    return ethCallAware;
  };
  const num = 10;
  const data = "0x2020";

  const tests = async (skipEthCallAware: boolean) => {
    it(`allows transaction to continue (skipped: ${skipEthCallAware})`, async () => {
      const ethCallAware = await setupEthCallAware();
      console.log((await ethCallAware.provider.getGasPrice()).toString())
      const res = await ethCallAware.functions.testFunction(num, data, skipEthCallAware);
      const receipt = await res.wait();

      const event = ethCallAware.interface.parseLog(receipt.logs[0]).args as TxSuccessEvent["args"];
      expect(event.data, "data").to.eq(data);
      expect(event.num.toNumber(), "num").to.eq(num);
    });

    it(`call reverts with data (skipped: ${skipEthCallAware})`, async () => {
      const ethCallAware = await setupEthCallAware();

      if (skipEthCallAware) {
        // we expect this to succeed
        await ethCallAware.callStatic.testFunction(num, data, skipEthCallAware);
      } else {
        expect(
          ethCallAware.callStatic.testFunction(num, data, skipEthCallAware),
          "Error message"
        ).to.be.revertedWith(`CallAwareData("${data}")`);
      }
    });

    it(`estimate gas returns correct value (skipped: ${skipEthCallAware})`, async () => {
      const ethCallAware = await setupEthCallAware();
      const gasEstimate = await ethCallAware.estimateGas.testFunction(num, data, skipEthCallAware);

      const res = await ethCallAware.functions.testFunction(num, data, skipEthCallAware);
      const receipt = await res.wait();
      const event = ethCallAware.interface.parseLog(receipt.logs[0]).args as TxSuccessEvent["args"];
      expect(event.data, "data").to.eq(data);
      expect(event.num.toNumber(), "num").to.eq(num);
      expect(gasEstimate.toNumber(), "gas used").to.eq(receipt.gasUsed);
    });
  };

  tests(false);
  tests(true);
});
