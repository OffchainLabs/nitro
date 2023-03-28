import {
  L1ToL2MessageGasEstimator,
  L1ToL2MessageStatus,
  L1TransactionReceipt,
  L2Network,
} from '@arbitrum/sdk'
import { getBaseFee } from '@arbitrum/sdk/dist/lib/utils/lib'
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
  EthVault__factory,
  RollupCore__factory,
} from '../../build/types'
import { setupNetworks, sleep } from '../../scripts/testSetup'
import { applyAlias } from '../contract/utils'

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
let userL1Wallet: Wallet
let userL2Wallet: Wallet
let token: ERC20
let inbox: ERC20Inbox
const excessFeeRefundAddress = Wallet.createRandom().address
const callValueRefundAddress = Wallet.createRandom().address

describe('ArbERC20Rollup', () => {
  // setup providers and connect deployed contracts
  before(async function () {
    const { l2Network } = await setupNetworks(config.ethUrl, config.arbUrl)
    _l2Network = l2Network

    l1Provider = new JsonRpcProvider(config.ethUrl)
    l2Provider = new JsonRpcProvider(config.arbUrl)
    userL1Wallet = new ethers.Wallet(
      ethers.utils.sha256(ethers.utils.toUtf8Bytes('user_l1user')),
      l1Provider
    )
    userL2Wallet = new ethers.Wallet(userL1Wallet.privateKey, l2Provider)
    token = ERC20__factory.connect(_l2Network.nativeToken, l1Provider)
    inbox = ERC20Inbox__factory.connect(_l2Network.ethBridge.inbox, l1Provider)
  })

  it('should have deployed bridge contracts', async function () {
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
    // snapshot state before deposit
    const userL1TokenBalance = await token.balanceOf(userL1Wallet.address)
    const userL2Balance = await l2Provider.getBalance(userL2Wallet.address)
    const bridgeL1TokenBalance = await token.balanceOf(
      _l2Network.ethBridge.bridge
    )

    /// deposit 25 tokens
    const amountToDeposit = ethers.utils.parseEther('25')
    await (
      await token
        .connect(userL1Wallet)
        .approve(_l2Network.ethBridge.bridge, amountToDeposit)
    ).wait()
    const depositTx = await inbox
      .connect(userL1Wallet)
      .depositERC20(amountToDeposit)

    // wait for deposit to be processed
    const depositRec = await L1TransactionReceipt.monkeyPatchEthDepositWait(
      depositTx
    ).wait()
    const l2Result = await depositRec.waitForL2(l2Provider)
    expect(l2Result.complete).to.be.true

    // check user balance increased on L2 and decreased on L1
    const userL1TokenBalanceAfter = await token.balanceOf(userL1Wallet.address)
    expect(userL1TokenBalance.sub(userL1TokenBalanceAfter)).to.be.eq(
      amountToDeposit
    )
    const userL2BalanceAfter = await l2Provider.getBalance(userL2Wallet.address)
    expect(userL2BalanceAfter.sub(userL2Balance)).to.be.eq(amountToDeposit)

    const bridgeL1TokenBalanceAfter = await token.balanceOf(
      _l2Network.ethBridge.bridge
    )
    // bridge escrow increased
    expect(bridgeL1TokenBalanceAfter.sub(bridgeL1TokenBalance)).to.be.eq(
      amountToDeposit
    )
  })

  it('should issue retryable ticket (no calldata)', async function () {
    // snapshot state before issuing retryable
    const userL1TokenBalance = await token.balanceOf(userL1Wallet.address)
    const userL1Balance = await l1Provider.getBalance(userL1Wallet.address)
    const userL2Balance = await l2Provider.getBalance(userL2Wallet.address)
    const aliasL2Balance = await l2Provider.getBalance(
      applyAlias(userL2Wallet.address)
    )
    const bridgeL1TokenBalance = await token.balanceOf(
      _l2Network.ethBridge.bridge
    )
    const excessFeeReceiverBalance = await l2Provider.getBalance(
      excessFeeRefundAddress
    )
    const callValueRefundReceiverBalance = await l2Provider.getBalance(
      callValueRefundAddress
    )

    //// retryables params

    const to = userL1Wallet.address
    const l2CallValue = ethers.utils.parseEther('37')
    const data = '0x'

    const l1ToL2MessageGasEstimate = new L1ToL2MessageGasEstimator(l2Provider)
    const retryableParams = await l1ToL2MessageGasEstimate.estimateAll(
      {
        from: userL1Wallet.address,
        to: to,
        l2CallValue: l2CallValue,
        excessFeeRefundAddress: excessFeeRefundAddress,
        callValueRefundAddress: callValueRefundAddress,
        data: data,
      },
      await getBaseFee(l1Provider),
      l1Provider
    )

    const tokenTotalFeeAmount = retryableParams.deposit
    const gasLimit = retryableParams.gasLimit
    const maxFeePerGas = retryableParams.maxFeePerGas
    const maxSubmissionCost = retryableParams.maxSubmissionCost

    /// deposit 37 tokens using retryable
    await (
      await token
        .connect(userL1Wallet)
        .approve(_l2Network.ethBridge.bridge, tokenTotalFeeAmount)
    ).wait()

    const retryableTx = await inbox
      .connect(userL1Wallet)
      .createRetryableTicket(
        to,
        l2CallValue,
        maxSubmissionCost,
        excessFeeRefundAddress,
        callValueRefundAddress,
        gasLimit,
        maxFeePerGas,
        tokenTotalFeeAmount,
        data
      )

    // wait for L2 msg to be executed
    await waitOnL2Msg(retryableTx)

    // check balances after retryable is processed
    const userL1TokenAfter = await token.balanceOf(userL1Wallet.address)
    expect(userL1TokenBalance.sub(userL1TokenAfter)).to.be.eq(
      tokenTotalFeeAmount
    )

    const userL2After = await l2Provider.getBalance(userL2Wallet.address)
    expect(userL2After.sub(userL2Balance)).to.be.eq(l2CallValue)

    const aliasL2BalanceAfter = await l2Provider.getBalance(
      applyAlias(userL2Wallet.address)
    )
    expect(aliasL2BalanceAfter).to.be.eq(aliasL2Balance)

    const excessFeeReceiverBalanceAfter = await l2Provider.getBalance(
      excessFeeRefundAddress
    )
    expect(excessFeeReceiverBalanceAfter).to.be.gte(excessFeeReceiverBalance)

    const callValueRefundReceiverBalanceAfter = await l2Provider.getBalance(
      callValueRefundAddress
    )
    expect(callValueRefundReceiverBalanceAfter).to.be.eq(
      callValueRefundReceiverBalance
    )

    const bridgeL1TokenAfter = await token.balanceOf(
      _l2Network.ethBridge.bridge
    )
    expect(bridgeL1TokenAfter.sub(bridgeL1TokenBalance)).to.be.eq(
      tokenTotalFeeAmount
    )
  })

  it('should issue retryable ticket', async function () {
    // deploy contract on L2 which will be retryable's target
    const ethVaultContract = await new EthVault__factory(
      userL2Wallet.connect(l2Provider)
    ).deploy()
    await ethVaultContract.deployed()

    // snapshot state before retryable
    const userL1TokenBalance = await token.balanceOf(userL1Wallet.address)
    const userL2Balance = await l2Provider.getBalance(userL2Wallet.address)
    const aliasL2Balance = await l2Provider.getBalance(
      applyAlias(userL2Wallet.address)
    )
    const bridgeL1TokenBalance = await token.balanceOf(
      _l2Network.ethBridge.bridge
    )
    const excessFeeReceiverBalance = await l2Provider.getBalance(
      excessFeeRefundAddress
    )
    const callValueRefundReceiverBalance = await l2Provider.getBalance(
      callValueRefundAddress
    )

    //// retryables params

    const to = ethVaultContract.address
    const l2CallValue = ethers.utils.parseEther('45')
    // calldata -> change 'version' field to 11
    const newValue = 11
    const data = new ethers.utils.Interface([
      'function setVersion(uint256 _version)',
    ]).encodeFunctionData('setVersion', [newValue])

    const l1ToL2MessageGasEstimate = new L1ToL2MessageGasEstimator(l2Provider)
    const retryableParams = await l1ToL2MessageGasEstimate.estimateAll(
      {
        from: userL1Wallet.address,
        to: to,
        l2CallValue: l2CallValue,
        excessFeeRefundAddress: excessFeeRefundAddress,
        callValueRefundAddress: callValueRefundAddress,
        data: data,
      },
      await getBaseFee(l1Provider),
      l1Provider
    )

    const tokenTotalFeeAmount = retryableParams.deposit
    const gasLimit = retryableParams.gasLimit
    const maxFeePerGas = retryableParams.maxFeePerGas
    const maxSubmissionCost = retryableParams.maxSubmissionCost

    /// execute retryable
    await (
      await token
        .connect(userL1Wallet)
        .approve(_l2Network.ethBridge.bridge, tokenTotalFeeAmount)
    ).wait()

    const retryableTx = await inbox
      .connect(userL1Wallet)
      .createRetryableTicket(
        to,
        l2CallValue,
        maxSubmissionCost,
        excessFeeRefundAddress,
        callValueRefundAddress,
        gasLimit,
        maxFeePerGas,
        tokenTotalFeeAmount,
        data
      )

    // wait for L2 msg to be executed
    await waitOnL2Msg(retryableTx)

    // check balances after retryable is processed
    const userL1TokenAfter = await token.balanceOf(userL2Wallet.address)
    expect(userL1TokenBalance.sub(userL1TokenAfter)).to.be.eq(
      tokenTotalFeeAmount
    )

    const userL2After = await l2Provider.getBalance(userL2Wallet.address)
    expect(userL2After).to.be.eq(userL2Balance)

    const ethVaultBalanceAfter = await l2Provider.getBalance(
      ethVaultContract.address
    )
    expect(ethVaultBalanceAfter).to.be.eq(l2CallValue)

    const ethVaultVersion = await ethVaultContract.version()
    expect(ethVaultVersion).to.be.eq(newValue)

    const aliasL2BalanceAfter = await l2Provider.getBalance(
      applyAlias(userL1Wallet.address)
    )
    expect(aliasL2BalanceAfter).to.be.eq(aliasL2Balance)

    const excessFeeReceiverBalanceAfter = await l2Provider.getBalance(
      excessFeeRefundAddress
    )
    expect(excessFeeReceiverBalanceAfter).to.be.gte(excessFeeReceiverBalance)

    const callValueRefundReceiverBalanceAfter = await l2Provider.getBalance(
      callValueRefundAddress
    )
    expect(callValueRefundReceiverBalanceAfter).to.be.eq(
      callValueRefundReceiverBalance
    )

    const bridgeL1TokenAfter = await token.balanceOf(
      _l2Network.ethBridge.bridge
    )
    expect(bridgeL1TokenAfter.sub(bridgeL1TokenBalance)).to.be.eq(
      tokenTotalFeeAmount
    )
  })
})

async function waitOnL2Msg(tx: ethers.ContractTransaction) {
  const retryableReceipt = await tx.wait()
  const l1TxReceipt = new L1TransactionReceipt(retryableReceipt)
  const messages = await l1TxReceipt.getL1ToL2Messages(l2Provider)

  // 1 msg expected
  const messageResult = await messages[0].waitForStatus()
  const status = messageResult.status
  expect(status).to.be.eq(L1ToL2MessageStatus.REDEEMED)
}
