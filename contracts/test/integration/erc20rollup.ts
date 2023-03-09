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
import { L1Network, L2Network } from '@arbitrum/sdk'
import { JsonRpcProvider } from '@ethersproject/providers'
import { expect } from 'chai'
import dotenv from 'dotenv'
import { ethers, Wallet } from 'ethers'
import {
  ERC20,
  ERC20Bridge__factory,
  ERC20Inbox,
  ERC20Inbox__factory,
  ERC20__factory,
  RollupCore__factory,
} from '../../build/types'
import { setupNetworks, sleep } from '../../scripts/testSetup'

dotenv.config()

export const config = {
  arbUrl: process.env['ARB_URL'] as string,
  ethUrl: process.env['ETH_URL'] as string,

  arbKey: process.env['ARB_KEY'] as string,
  ethKey: process.env['ETH_KEY'] as string,
}

let l1Provider: JsonRpcProvider
let l2Provider: JsonRpcProvider
let _l2Network: L2Network & { nativeToken: string }
let user: Wallet
let token: ERC20
let inbox: ERC20Inbox

describe('ArbERC20Rollup', () => {
  before(async function () {
    const { l2Network } = await setupNetworks(config.ethUrl, config.arbUrl)
    _l2Network = l2Network
    l1Provider = new JsonRpcProvider(config.ethUrl)
    l2Provider = new JsonRpcProvider(config.arbUrl)
    user = new ethers.Wallet(
      ethers.utils.sha256(ethers.utils.toUtf8Bytes('user_l1user')),
      l1Provider
    )
    token = ERC20__factory.connect(_l2Network.nativeToken, l1Provider)
    inbox = ERC20Inbox__factory.connect(_l2Network.ethBridge.inbox, l1Provider)
  })

  it('should deploy bridge contracts', async function () {
    // get rollup as entry point
    const rollup = RollupCore__factory.connect(
      _l2Network.ethBridge.rollup,
      l1Provider
    )

    // check contract refs are properly set
    expect(rollup.address).to.be.eq(_l2Network.ethBridge.rollup)
    expect((await rollup.sequencerInbox()).toLowerCase()).to.be.eq(
      _l2Network.ethBridge.sequencerInbox
    )
    expect(await rollup.outbox()).to.be.eq(_l2Network.ethBridge.outbox)
    expect((await rollup.inbox()).toLowerCase()).to.be.eq(
      _l2Network.ethBridge.inbox
    )

    const erc20Bridge = ERC20Bridge__factory.connect(
      await rollup.bridge(),
      l1Provider
    )
    expect(erc20Bridge.address.toLowerCase()).to.be.eq(
      _l2Network.ethBridge.bridge
    )
    expect((await erc20Bridge.nativeToken()).toLowerCase()).to.be.eq(
      _l2Network.nativeToken
    )
  })

  it('should deposit native token to L2', async function () {
    const userL1TokenBalance = await token.balanceOf(user.address)
    const userL2Balance = await l2Provider.getBalance(user.address)
    const bridgeL1TokenBalance = await token.balanceOf(
      _l2Network.ethBridge.bridge
    )

    /// deposit 25 tokens
    const amountToDeposit = ethers.utils.parseEther('25')
    await (
      await token
        .connect(user)
        .approve(_l2Network.ethBridge.bridge, amountToDeposit)
    ).wait()
    await (await inbox.connect(user).depositERC20(amountToDeposit)).wait()

    // sleep 20s (time for deposit to be processed)
    await sleep(20000)

    // check user balance increased on L2 and decreased on L1
    const userL1TokenBalanceAfter = await token.balanceOf(user.address)
    expect(userL1TokenBalance.sub(userL1TokenBalanceAfter)).to.be.eq(
      amountToDeposit
    )
    const userL2BalanceAfter = await l2Provider.getBalance(user.address)
    expect(userL2BalanceAfter.sub(userL2Balance)).to.be.eq(amountToDeposit)

    const bridgeL1TokenBalanceAfter = await token.balanceOf(
      _l2Network.ethBridge.bridge
    )
    // bridge escrow increased
    expect(bridgeL1TokenBalanceAfter.sub(bridgeL1TokenBalance)).to.be.eq(
      amountToDeposit
    )
  })
})
