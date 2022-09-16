import { ethers } from 'hardhat'
import { expect } from 'chai'
import TestCase from './outbox/withdraw-testcase.json'
import { BigNumber, Contract, ContractFactory, Signer } from 'ethers'
import { TransparentUpgradeableProxy__factory } from '../../build/types/factories/@openzeppelin/contracts/proxy/transparent'

async function sendEth(
  send_account: string,
  to_address: string,
  send_token_amount: BigNumber
) {
  const nonce = await ethers.provider.getTransactionCount(
    send_account,
    'latest'
  )
  const gas_price = await ethers.provider.getGasPrice()

  const tx = {
    from: send_account,
    to: to_address,
    value: send_token_amount,
    nonce: nonce,
    gasLimit: 100000, // 100000
    gasPrice: gas_price,
  }
  const signer = ethers.provider.getSigner(send_account)
  await signer.sendTransaction(tx)
}

async function setSendRoot(cases: any, outbox: Contract, signer: Signer) {
  const length = cases.length
  for (let i = 0; i < length; i++) {
    await outbox
      .connect(signer)
      .updateSendRoot(cases[i].root, cases[i].l2blockhash)
  }
}

const deployBehindProxy = async <T extends ContractFactory>(
  deployer: Signer,
  factory: T,
  admin: string,
  dataToCallProxy = '0x'
): Promise<ReturnType<T['deploy']>> => {
  const instance = await factory.connect(deployer).deploy()
  await instance.deployed()
  const proxy = await new TransparentUpgradeableProxy__factory()
    .connect(deployer)
    .deploy(instance.address, admin, dataToCallProxy)
  await proxy.deployed()
  return instance.attach(proxy.address)
}

describe('Outbox', async function () {
  let outboxWithOpt: Contract
  let outboxWithoutOpt: Contract
  let bridge: Contract
  const cases = TestCase.cases
  const sentEthAmount = ethers.utils.parseEther('10')
  let accounts: Signer[]
  let rollup: Signer

  before(async function () {
    accounts = await ethers.getSigners()
    const OutboxWithOpt = await ethers.getContractFactory('Outbox')
    const OutboxWithoutOpt = await ethers.getContractFactory(
      'OutboxWithoutOptTester'
    )
    const Bridge = await ethers.getContractFactory('BridgeTester')
    outboxWithOpt = await deployBehindProxy(
      accounts[0],
      OutboxWithOpt,
      await accounts[1].getAddress()
    )
    rollup = accounts[3]
    outboxWithoutOpt = await OutboxWithoutOpt.deploy()
    bridge = (await Bridge.deploy()).connect(rollup)
    await bridge.initialize(await rollup.getAddress())
    await outboxWithOpt.initialize(bridge.address)
    await outboxWithoutOpt.initialize(bridge.address)
    await bridge.setOutbox(outboxWithOpt.address, true)
    await bridge.setOutbox(outboxWithoutOpt.address, true)
    await setSendRoot(cases, outboxWithOpt, rollup)
    await setSendRoot(cases, outboxWithoutOpt, rollup)
    await sendEth(await accounts[0].getAddress(), bridge.address, sentEthAmount)
  })

  it('First call to initial some storage', async function () {
    await sendEth(await accounts[0].getAddress(), cases[0].to, sentEthAmount)
    await expect(
      outboxWithOpt.executeTransaction(
        cases[0].proof,
        cases[0].index,
        cases[0].l2Sender,
        cases[0].to,
        cases[0].l2Block,
        cases[0].l1Block,
        cases[0].l2Timestamp,
        cases[0].value,
        cases[0].data
      )
    ).to.emit(bridge, 'BridgeCallTriggered')
    await expect(
      outboxWithoutOpt.executeTransaction(
        cases[0].proof,
        cases[0].index,
        cases[0].l2Sender,
        cases[0].to,
        cases[0].l2Block,
        cases[0].l1Block,
        cases[0].l2Timestamp,
        cases[0].value,
        cases[0].data
      )
    ).to.emit(bridge, 'BridgeCallTriggered')
    //await outboxWithOpt.executeTransaction(cases[0].proof,cases[0].index,cases[0].l2Sender,cases[0].to,cases[0].l2Block,cases[0].l1Block,cases[0].l2Timestamp,cases[0].value,cases[0].data);
  })

  it('Call twice without storage initail cost', async function () {
    await sendEth(await accounts[0].getAddress(), cases[1].to, sentEthAmount)
    await expect(
      outboxWithOpt.executeTransaction(
        cases[1].proof,
        cases[1].index,
        cases[1].l2Sender,
        cases[1].to,
        cases[1].l2Block,
        cases[1].l1Block,
        cases[1].l2Timestamp,
        cases[1].value,
        cases[1].data
      )
    ).to.emit(bridge, 'BridgeCallTriggered')
    await expect(
      outboxWithoutOpt.executeTransaction(
        cases[1].proof,
        cases[1].index,
        cases[1].l2Sender,
        cases[1].to,
        cases[1].l2Block,
        cases[1].l1Block,
        cases[1].l2Timestamp,
        cases[1].value,
        cases[1].data
      )
    ).to.emit(bridge, 'BridgeCallTriggered')
  })

  it('third call', async function () {
    await sendEth(await accounts[0].getAddress(), cases[2].to, sentEthAmount)
    await expect(
      outboxWithOpt.executeTransaction(
        cases[2].proof,
        cases[2].index,
        cases[2].l2Sender,
        cases[2].to,
        cases[2].l2Block,
        cases[2].l1Block,
        cases[2].l2Timestamp,
        cases[2].value,
        cases[2].data
      )
    ).to.emit(bridge, 'BridgeCallTriggered')
    await expect(
      outboxWithoutOpt.executeTransaction(
        cases[2].proof,
        cases[2].index,
        cases[2].l2Sender,
        cases[2].to,
        cases[2].l2Block,
        cases[2].l1Block,
        cases[2].l2Timestamp,
        cases[2].value,
        cases[2].data
      )
    ).to.emit(bridge, 'BridgeCallTriggered')
  })
})
