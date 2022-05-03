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
import { CallOverrides } from "ethers";

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

  it(`estimate gas returns correct value`, async () => {
    const ethCallAware = await setupEthCallAware();
    const gasEstimate = await ethCallAware.estimateGas.testFunction(num, data);

    const res = await ethCallAware.functions.testFunction(num, data);
    const receipt = await res.wait();
    const event = ethCallAware.interface.parseLog(receipt.logs[0])
      .args as TxSuccessEvent["args"];
    expect(event.data, "data").to.eq(data);
    expect(event.num.toNumber(), "num").to.eq(num);
    expect(gasEstimate.toNumber(), "gas used").to.eq(receipt.gasUsed);
  });

  for (let i = 0; i < 2; i++) {
    const opts: CallOverrides = {};

    if (i == 0) opts["gasPrice"] = ethers.BigNumber.from("0xcA11");
    else opts["from"] = "0x000000000000000000000000000000000000cA11";

    describe(`running tests by overloading ${i === 0 ? "gasPrice" : "txOrigin"}`, () => {
      it(`allows transaction to continue`, async () => {
        const ethCallAware = await setupEthCallAware();
        const res = await ethCallAware.functions.testFunction(num, data);
        const receipt = await res.wait();

        const event = ethCallAware.interface.parseLog(receipt.logs[0])
          .args as TxSuccessEvent["args"];
        expect(event.data, "data").to.eq(data);
        expect(event.num.toNumber(), "num").to.eq(num);
      });

      it(`call reverts with data`, async () => {
        const ethCallAware = await setupEthCallAware();

        expect(
          ethCallAware.callStatic.testFunction(num, data, opts),
          "Error message"
        ).to.be.revertedWith(`CallAwareData(0, "${data}")`);
      });
    });
  }
});
