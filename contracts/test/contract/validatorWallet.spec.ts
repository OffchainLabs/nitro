import { expect } from 'chai'
import { ethers } from 'hardhat'
import {
  ValidatorWalletCreator,
  ValidatorWalletCreator__factory,
  ValidatorWallet,
  RollupMock,
} from '../../build/types'
import { initializeAccounts } from './utils'

type ArrayElement<A> = A extends readonly (infer T)[] ? T : never

describe('Validator Wallet', () => {
  let accounts: Awaited<ReturnType<typeof initializeAccounts>>
  let owner: ArrayElement<typeof accounts>
  let executor: ArrayElement<typeof accounts>
  let walletCreator: ValidatorWalletCreator
  let wallet: ValidatorWallet
  let rollupMock1: RollupMock
  let rollupMock2: RollupMock

  before(async () => {
    accounts = await initializeAccounts()
    const WalletCreator: ValidatorWalletCreator__factory =
      await ethers.getContractFactory('ValidatorWalletCreator')
    walletCreator = await WalletCreator.deploy()
    await walletCreator.deployed()

    owner = await accounts[0]
    executor = await accounts[1]
    const walletCreationTx = await (await walletCreator.createWallet([])).wait()

    const events = walletCreationTx.logs
      .filter(
        curr =>
          curr.topics[0] ===
          walletCreator.interface.getEventTopic('WalletCreated')
      )
      .map(curr => walletCreator.interface.parseLog(curr))
    if (events.length !== 1) throw new Error('No Events!')
    const walletAddr = events[0].args.walletAddress
    const Wallet = await ethers.getContractFactory('ValidatorWallet')
    wallet = (await Wallet.attach(walletAddr)) as ValidatorWallet

    await wallet.setExecutor([await executor.getAddress()], [true])
    await wallet.transferOwnership(await owner.getAddress())

    const RollupMock = await ethers.getContractFactory('RollupMock')
    rollupMock1 = (await RollupMock.deploy()) as RollupMock
    rollupMock2 = (await RollupMock.deploy()) as RollupMock

    await accounts[0].sendTransaction({
      to: wallet.address,
      value: ethers.utils.parseEther('5'),
    })
  })

  it('should validate destination addresses', async function () {
    const addrs = [
      '0x1234567812345678123456781234567812345678',
      '0x0000000000000000000000000000000000000000',
      '0x0123000000000000000000000000000000000000',
    ]
    await wallet.setAllowedExecutorDestinations(addrs, [true, true, true])

    expect(await wallet.allowedExecutorDestinations(addrs[0])).to.be.true
    expect(await wallet.allowedExecutorDestinations(addrs[1])).to.be.true
    expect(await wallet.allowedExecutorDestinations(addrs[2])).to.be.true
    expect(
      await wallet.allowedExecutorDestinations(
        '0x1114567812345678123456781234567812341111'
      )
    ).to.be.false

    // should fail if random destination address on executor, but should work for the owner
    const rand = '0x1114567812345678123456781234567899941111'
    await expect(wallet.connect(executor).validateExecuteTransaction(rand)).to
      .be.reverted
    await wallet.connect(owner).validateExecuteTransaction(rand)

    for (const addr of addrs) {
      await wallet.connect(owner).validateExecuteTransaction(addr)
      await wallet.connect(executor).validateExecuteTransaction(addr)
      // TODO: update waffle once released with fix for custom errors https://github.com/TrueFiEng/Waffle/pull/719
      await expect(wallet.connect(executor).validateExecuteTransaction(rand)).to
        .be.reverted
    }
  })

  it('should not allow executor to execute certain txs', async function () {
    const data = rollupMock1.interface.encodeFunctionData('withdrawStakerFunds')

    await expect(
      wallet.connect(executor).executeTransaction(data, rollupMock1.address, 0)
    ).to.be.revertedWith(
      `OnlyOwnerDestination("${await owner.getAddress()}", "${await executor.getAddress()}", "${
        rollupMock1.address
      }")`
    )
    await expect(
      wallet.connect(owner).executeTransaction(data, rollupMock1.address, 0)
    ).to.emit(rollupMock1, 'WithdrawTriggered')

    await wallet.setAllowedExecutorDestinations([rollupMock1.address], [true])
    await expect(
      wallet.connect(executor).executeTransaction(data, rollupMock1.address, 0)
    ).to.emit(rollupMock1, 'WithdrawTriggered')
  })

  it('should reject batch if single tx is not allowed by executor', async function () {
    const data = [
      rollupMock1.interface.encodeFunctionData('removeOldZombies', [0]),
      rollupMock2.interface.encodeFunctionData('withdrawStakerFunds'),
    ]

    await wallet.setAllowedExecutorDestinations(
      [rollupMock1.address, rollupMock2.address],
      [true, false]
    )

    await expect(
      wallet
        .connect(executor)
        .executeTransactions(
          data,
          [rollupMock1.address, rollupMock2.address],
          [0, 0]
        )
    ).to.be.revertedWith(
      `OnlyOwnerDestination("${await owner.getAddress()}", "${await executor.getAddress()}", "${
        rollupMock2.address
      }")`
    )
  })
})
