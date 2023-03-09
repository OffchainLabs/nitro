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
import { JsonRpcProvider } from '@ethersproject/providers'
import { expect } from 'chai'
import dotenv from 'dotenv'
import { ERC20Bridge__factory, RollupCore__factory } from '../../build/types'
import { setupNetworks } from '../../scripts/testSetup'

dotenv.config()

export const config = {
  arbUrl: process.env['ARB_URL'] as string,
  ethUrl: process.env['ETH_URL'] as string,

  arbKey: process.env['ARB_KEY'] as string,
  ethKey: process.env['ETH_KEY'] as string,
}

describe('ArbERC20Rollup', () => {
  it('should deploy bridge contracts', async function () {
    // const { l1Signer, l2Signer, l1Deployer, l2Deployer } = await testSetup()

    const { l2Network } = await setupNetworks(config.ethUrl, config.arbUrl)

    const l1Provider = new JsonRpcProvider(config.ethUrl)

    // get rollup as entry point
    const rollup = RollupCore__factory.connect(
      l2Network.ethBridge.rollup,
      l1Provider
    )

    // check contract refs are properly set
    expect(rollup.address).to.be.eq(l2Network.ethBridge.rollup)
    expect((await rollup.sequencerInbox()).toLowerCase()).to.be.eq(
      l2Network.ethBridge.sequencerInbox
    )
    expect(await rollup.outbox()).to.be.eq(l2Network.ethBridge.outbox)
    expect((await rollup.inbox()).toLowerCase()).to.be.eq(
      l2Network.ethBridge.inbox
    )

    const erc20Bridge = ERC20Bridge__factory.connect(
      await rollup.bridge(),
      l1Provider
    )
    expect(erc20Bridge.address.toLowerCase()).to.be.eq(
      l2Network.ethBridge.bridge
    )
    expect((await erc20Bridge.nativeToken()).toLowerCase()).to.be.eq(
      l2Network.nativeToken
    )
  })
})
