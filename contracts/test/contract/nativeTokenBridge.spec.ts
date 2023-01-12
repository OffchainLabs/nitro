import { expect } from 'chai'
import { Signer } from 'ethers'
import { ethers } from 'hardhat'
import {
  NativeTokenBridge,
  NativeTokenBridge__factory,
  ProxyAdmin__factory,
  TransparentUpgradeableProxy__factory,
} from '../../build/types'
import { initializeAccounts } from './utils'

const rollup = '0x0000000000000000000000000000000000001336'
const nativeToken = '0x0000000000000000000000000000000000001337'

describe('NativeTokenBridge', () => {
  let nativeTokenBridge: NativeTokenBridge
  let admin: Signer

  before('deploy contracts', async function () {
    const accounts = await initializeAccounts()
    admin = accounts[0]

    // deploy logic
    const bridgeLogic = await new NativeTokenBridge__factory(admin).deploy()

    // deploy proxy
    const proxyAdmin = await new ProxyAdmin__factory(admin).deploy()
    const bridgeProxy = await new TransparentUpgradeableProxy__factory(
      admin
    ).deploy(bridgeLogic.address, proxyAdmin.address, '0x')

    // store ref
    nativeTokenBridge = NativeTokenBridge__factory.connect(
      bridgeProxy.address,
      admin
    )
  })

  it('should be initialized', async function () {
    await nativeTokenBridge['initialize(address,address)'](rollup, nativeToken)

    expect(await nativeTokenBridge.nativeToken()).to.eq(nativeToken)
    expect(await nativeTokenBridge.rollup()).to.eq(rollup)
    expect(await nativeTokenBridge.activeOutbox()).to.eq(
      ethers.constants.AddressZero
    )
  })
})
