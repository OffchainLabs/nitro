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

import { ethers, network } from 'hardhat'
import { BigNumber } from '@ethersproject/bignumber'
import { Block, TransactionReceipt } from '@ethersproject/providers'
import { expect } from 'chai'
import {
  Bridge,
  Bridge__factory,
  Inbox,
  Inbox__factory,
  MessageTester,
  RollupMock__factory,
  SequencerInbox,
  SequencerInbox__factory,
  TransparentUpgradeableProxy__factory,
} from '../../build/types'
import { applyAlias, initializeAccounts } from './utils'
import { Event } from '@ethersproject/contracts'
import { Interface } from '@ethersproject/abi'
import {
  BridgeInterface,
  MessageDeliveredEvent,
} from '../../build/types/src/bridge/Bridge'
import { Signer } from 'ethers'
import { Toolkit4844 } from './toolkit4844'
import { data } from './batchData.json'

const mineBlocks = async (count: number, timeDiffPerBlock = 14) => {
  const block = (await network.provider.send('eth_getBlockByNumber', [
    'latest',
    false,
  ])) as Block
  let timestamp = BigNumber.from(block.timestamp).toNumber()
  for (let i = 0; i < count; i++) {
    timestamp = timestamp + timeDiffPerBlock
    await network.provider.send('evm_mine', [timestamp])
  }
}

describe('SequencerInboxForceInclude', async () => {
  const findMatchingLogs = <TInterface extends Interface, TEvent extends Event>(
    receipt: TransactionReceipt,
    iFace: TInterface,
    eventTopicGen: (i: TInterface) => string
  ): TEvent['args'][] => {
    const logs = receipt.logs.filter(
      log => log.topics[0] === eventTopicGen(iFace)
    )
    return logs.map(l => iFace.parseLog(l).args as TEvent['args'])
  }

  const getMessageDeliveredEvents = (receipt: TransactionReceipt) => {
    const bridgeInterface = Bridge__factory.createInterface()
    return findMatchingLogs<BridgeInterface, MessageDeliveredEvent>(
      receipt,
      bridgeInterface,
      i => i.getEventTopic(i.getEvent('MessageDelivered'))
    )
  }

  const sendDelayedTx = async (
    sender: Signer,
    inbox: Inbox,
    bridge: Bridge,
    messageTester: MessageTester,
    l2Gas: number,
    l2GasPrice: number,
    nonce: number,
    destAddr: string,
    amount: BigNumber,
    data: string
  ) => {
    const countBefore = (
      await bridge.functions.delayedMessageCount()
    )[0].toNumber()
    const sendUnsignedTx = await inbox
      .connect(sender)
      .sendUnsignedTransaction(l2Gas, l2GasPrice, nonce, destAddr, amount, data)
    const sendUnsignedTxReceipt = await sendUnsignedTx.wait()

    const countAfter = (
      await bridge.functions.delayedMessageCount()
    )[0].toNumber()
    expect(countAfter, 'Unexpected inbox count').to.eq(countBefore + 1)

    const senderAddr = applyAlias(await sender.getAddress())

    const messageDeliveredEvent = getMessageDeliveredEvents(
      sendUnsignedTxReceipt
    )[0]
    const l1BlockNumber = sendUnsignedTxReceipt.blockNumber
    const blockL1 = await sender.provider!.getBlock(l1BlockNumber)
    const baseFeeL1 = blockL1.baseFeePerGas!.toNumber()
    const l1BlockTimestamp = blockL1.timestamp
    const delayedAcc = await bridge.delayedInboxAccs(countBefore)

    // need to hex pad the address
    const messageDataHash = ethers.utils.solidityKeccak256(
      ['uint8', 'uint256', 'uint256', 'uint256', 'uint256', 'uint256', 'bytes'],
      [
        0,
        l2Gas,
        l2GasPrice,
        nonce,
        ethers.utils.hexZeroPad(destAddr, 32),
        amount,
        data,
      ]
    )
    expect(
      messageDeliveredEvent.messageDataHash,
      'Incorrect messageDataHash'
    ).to.eq(messageDataHash)

    const messageHash = (
      await messageTester.functions.messageHash(
        3,
        senderAddr,
        l1BlockNumber,
        l1BlockTimestamp,
        countBefore,
        baseFeeL1,
        messageDataHash
      )
    )[0]

    const prevAccumulator = messageDeliveredEvent.beforeInboxAcc
    expect(prevAccumulator, 'Incorrect prev accumulator').to.eq(
      countBefore === 0
        ? ethers.utils.hexZeroPad('0x', 32)
        : await bridge.delayedInboxAccs(countBefore - 1)
    )

    const nextAcc = (
      await messageTester.functions.accumulateInboxMessage(
        prevAccumulator,
        messageHash
      )
    )[0]

    expect(delayedAcc, 'Incorrect delayed acc').to.eq(nextAcc)

    return {
      baseFeeL1: baseFeeL1,
      deliveredMessageEvent: messageDeliveredEvent,
      l1BlockNumber,
      l1BlockTimestamp,
      delayedAcc,
      l2Gas,
      l2GasPrice,
      nonce,
      destAddr,
      amount,
      data,
      senderAddr,
      inboxAccountLength: countAfter,
    }
  }

  const forceIncludeMessages = async (
    sequencerInbox: SequencerInbox,
    newTotalDelayedMessagesRead: number,
    kind: number,
    l1blockNumber: number,
    l1Timestamp: number,
    l1BaseFee: number,
    senderAddr: string,
    messageDataHash: string,
    expectedErrorType?: string
  ) => {
    const inboxLengthBefore = (await sequencerInbox.batchCount()).toNumber()

    const forceInclusionTx = sequencerInbox.forceInclusion(
      newTotalDelayedMessagesRead,
      kind,
      [l1blockNumber, l1Timestamp],
      l1BaseFee,
      senderAddr,
      messageDataHash
    )
    if (expectedErrorType) {
      await expect(forceInclusionTx).to.be.revertedWith(expectedErrorType)
    } else {
      await (await forceInclusionTx).wait()

      const totalDelayedMessagsReadAfter = (
        await sequencerInbox.totalDelayedMessagesRead()
      ).toNumber()
      expect(
        totalDelayedMessagsReadAfter,
        'Incorrect totalDelayedMessagesRead after.'
      ).to.eq(newTotalDelayedMessagesRead)
      const inboxLengthAfter = (await sequencerInbox.batchCount()).toNumber()
      expect(
        inboxLengthAfter - inboxLengthBefore,
        'Inbox not incremented'
      ).to.eq(1)
    }
  }

  const setupSequencerInbox = async (maxDelayBlocks = 10, maxDelayTime = 0) => {
    const accounts = await initializeAccounts()
    const admin = accounts[0]
    const adminAddr = await admin.getAddress()
    const user = accounts[1]
    // const dummyRollup = accounts[2]
    const rollupOwner = accounts[3]
    const batchPoster = accounts[4]
    // const batchPosterManager = accounts[5]

    const rollupMockFac = (await ethers.getContractFactory(
      'RollupMock'
    )) as RollupMock__factory
    const rollup = await rollupMockFac.deploy(await rollupOwner.getAddress())

    const reader4844 = await Toolkit4844.deployReader4844(admin)
    const sequencerInboxFac = (await ethers.getContractFactory(
      'SequencerInbox'
    )) as SequencerInbox__factory
    const seqInboxTemplate = await sequencerInboxFac.deploy(
      117964,
      reader4844.address,
      false,
      false
    )
    const inboxFac = (await ethers.getContractFactory(
      'Inbox'
    )) as Inbox__factory
    const inboxTemplate = await inboxFac.deploy(117964)
    const bridgeFac = (await ethers.getContractFactory(
      'Bridge'
    )) as Bridge__factory
    const bridgeTemplate = await bridgeFac.deploy()
    const transparentUpgradeableProxyFac = (await ethers.getContractFactory(
      'TransparentUpgradeableProxy'
    )) as TransparentUpgradeableProxy__factory

    const bridgeProxy = await transparentUpgradeableProxyFac.deploy(
      bridgeTemplate.address,
      adminAddr,
      '0x'
    )
    const sequencerInboxProxy = await transparentUpgradeableProxyFac.deploy(
      seqInboxTemplate.address,
      adminAddr,
      '0x'
    )
    const inboxProxy = await transparentUpgradeableProxyFac.deploy(
      inboxTemplate.address,
      adminAddr,
      '0x'
    )
    const bridge = await bridgeFac.attach(bridgeProxy.address).connect(user)
    const bridgeAdmin = await bridgeFac
      .attach(bridgeProxy.address)
      .connect(rollupOwner)
    const sequencerInbox = await sequencerInboxFac
      .attach(sequencerInboxProxy.address)
      .connect(user)
    await bridge.initialize(rollup.address)

    await sequencerInbox.initialize(
      bridgeProxy.address,
      {
        delayBlocks: maxDelayBlocks,
        delaySeconds: maxDelayTime,
        futureBlocks: 10,
        futureSeconds: 3000,
      },
      {
        threshold: 0,
        max: 0,
        replenishRateInBasis: 0,
      }
    )

    await (
      await sequencerInbox
        .connect(rollupOwner)
        .setIsBatchPoster(await batchPoster.getAddress(), true)
    ).wait()

    const inbox = await inboxFac.attach(inboxProxy.address).connect(user)

    await inbox.initialize(bridgeProxy.address, sequencerInbox.address)

    await bridgeAdmin.setDelayedInbox(inbox.address, true)
    await bridgeAdmin.setSequencerInbox(sequencerInbox.address)

    await (
      await sequencerInbox
        .connect(rollupOwner)
        .setIsBatchPoster(await batchPoster.getAddress(), true)
    ).wait()

    const messageTester = (await (
      await ethers.getContractFactory('MessageTester')
    ).deploy()) as MessageTester

    return {
      user,
      bridge: bridge,
      inbox: inbox,
      sequencerInbox: sequencerInbox as SequencerInbox,
      messageTester,
      inboxProxy,
      inboxTemplate,
      batchPoster,
      bridgeProxy,
      rollup,
      rollupOwner,
    }
  }

  it('can add batch', async () => {
    const { user, inbox, bridge, messageTester, sequencerInbox, batchPoster } =
      await setupSequencerInbox()

    await sendDelayedTx(
      user,
      inbox,
      bridge,
      messageTester,
      1000000,
      21000000000,
      0,
      await user.getAddress(),
      BigNumber.from(10),
      '0x1010'
    )

    const messagesRead = await bridge.delayedMessageCount()
    const seqReportedMessageSubCount =
      await bridge.sequencerReportedSubMessageCount()
    await (
      await sequencerInbox
        .connect(batchPoster)
        .functions[
          'addSequencerL2BatchFromOrigin(uint256,bytes,uint256,address,uint256,uint256)'
        ](
          0,
          data,
          messagesRead,
          ethers.constants.AddressZero,
          seqReportedMessageSubCount,
          seqReportedMessageSubCount.add(10),
          { gasLimit: 10000000 }
        )
    ).wait()
  })

  it('can force-include', async () => {
    const { user, inbox, bridge, messageTester, sequencerInbox } =
      await setupSequencerInbox()

    const delayedTx = await sendDelayedTx(
      user,
      inbox,
      bridge,
      messageTester,
      1000000,
      21000000000,
      0,
      await user.getAddress(),
      BigNumber.from(10),
      '0x1010'
    )

    const [delayBlocks, , ,] = await sequencerInbox.maxTimeVariation()
    await mineBlocks(delayBlocks.toNumber())

    await forceIncludeMessages(
      sequencerInbox,
      delayedTx.inboxAccountLength,
      delayedTx.deliveredMessageEvent.kind,
      delayedTx.l1BlockNumber,
      delayedTx.l1BlockTimestamp,
      delayedTx.baseFeeL1,
      delayedTx.senderAddr,
      delayedTx.deliveredMessageEvent.messageDataHash
    )
  })

  it('can force-include-with-max-seqReportedCount', async () => {
    const { user, inbox, bridge, messageTester, batchPoster, sequencerInbox } =
      await setupSequencerInbox()

    await sequencerInbox
      .connect(batchPoster)
      [
        'addSequencerL2BatchFromOrigin(uint256,bytes,uint256,address,uint256,uint256)'
      ](
        0,
        '0x',
        0,
        ethers.constants.AddressZero,
        0,
        ethers.constants.MaxUint256
      )

    const delayedTx = await sendDelayedTx(
      user,
      inbox,
      bridge,
      messageTester,
      1000000,
      21000000000,
      0,
      await user.getAddress(),
      BigNumber.from(10),
      '0x1010'
    )
    const maxTimeVariation = await sequencerInbox.maxTimeVariation()

    await mineBlocks(maxTimeVariation[0].toNumber())

    await forceIncludeMessages(
      sequencerInbox,
      delayedTx.inboxAccountLength,
      delayedTx.deliveredMessageEvent.kind,
      delayedTx.l1BlockNumber,
      delayedTx.l1BlockTimestamp,
      delayedTx.baseFeeL1,
      delayedTx.senderAddr,
      delayedTx.deliveredMessageEvent.messageDataHash
    )
  })

  it('can force-include one after another', async () => {
    const { user, inbox, bridge, messageTester, sequencerInbox } =
      await setupSequencerInbox()
    const delayedTx = await sendDelayedTx(
      user,
      inbox,
      bridge,
      messageTester,
      1000000,
      21000000000,
      0,
      await user.getAddress(),
      BigNumber.from(10),
      '0x1010'
    )

    const delayedTx2 = await sendDelayedTx(
      user,
      inbox,
      bridge,
      messageTester,
      1000000,
      21000000000,
      1,
      await user.getAddress(),
      BigNumber.from(10),
      '0xdeadface'
    )

    const [delayBlocks, , ,] = await sequencerInbox.maxTimeVariation()
    await mineBlocks(delayBlocks.toNumber())

    await forceIncludeMessages(
      sequencerInbox,
      delayedTx.inboxAccountLength,
      delayedTx.deliveredMessageEvent.kind,
      delayedTx.l1BlockNumber,
      delayedTx.l1BlockTimestamp,
      delayedTx.baseFeeL1,
      delayedTx.senderAddr,
      delayedTx.deliveredMessageEvent.messageDataHash
    )
    await forceIncludeMessages(
      sequencerInbox,
      delayedTx2.inboxAccountLength,
      delayedTx2.deliveredMessageEvent.kind,
      delayedTx2.l1BlockNumber,
      delayedTx2.l1BlockTimestamp,
      delayedTx2.baseFeeL1,
      delayedTx2.senderAddr,
      delayedTx2.deliveredMessageEvent.messageDataHash
    )
  })

  it('can force-include three at once', async () => {
    const { user, inbox, bridge, messageTester, sequencerInbox } =
      await setupSequencerInbox()
    await sendDelayedTx(
      user,
      inbox,
      bridge,
      messageTester,
      1000000,
      21000000000,
      0,
      await user.getAddress(),
      BigNumber.from(10),
      '0x1010'
    )
    await sendDelayedTx(
      user,
      inbox,
      bridge,
      messageTester,
      1000000,
      21000000000,
      1,
      await user.getAddress(),
      BigNumber.from(10),
      '0x101010'
    )
    const delayedTx3 = await sendDelayedTx(
      user,
      inbox,
      bridge,
      messageTester,
      1000000,
      21000000000,
      10,
      await user.getAddress(),
      BigNumber.from(10),
      '0x10101010'
    )

    const [delayBlocks, , ,] = await sequencerInbox.maxTimeVariation()
    await mineBlocks(delayBlocks.toNumber())

    await forceIncludeMessages(
      sequencerInbox,
      delayedTx3.inboxAccountLength,
      delayedTx3.deliveredMessageEvent.kind,
      delayedTx3.l1BlockNumber,
      delayedTx3.l1BlockTimestamp,
      delayedTx3.baseFeeL1,
      delayedTx3.senderAddr,
      delayedTx3.deliveredMessageEvent.messageDataHash
    )
  })

  it('cannot include before max block delay', async () => {
    const { user, inbox, bridge, messageTester, sequencerInbox } =
      await setupSequencerInbox(10, 100)
    const delayedTx = await sendDelayedTx(
      user,
      inbox,
      bridge,
      messageTester,
      1000000,
      21000000000,
      0,
      await user.getAddress(),
      BigNumber.from(10),
      '0x1010'
    )

    const [delayBlocks, , ,] = await sequencerInbox.maxTimeVariation()
    await mineBlocks(delayBlocks.toNumber() - 1, 5)

    await forceIncludeMessages(
      sequencerInbox,
      delayedTx.inboxAccountLength,
      delayedTx.deliveredMessageEvent.kind,
      delayedTx.l1BlockNumber,
      delayedTx.l1BlockTimestamp,
      delayedTx.baseFeeL1,
      delayedTx.senderAddr,
      delayedTx.deliveredMessageEvent.messageDataHash,
      'ForceIncludeBlockTooSoon'
    )
  })

  it('should fail to call sendL1FundedUnsignedTransactionToFork', async function () {
    const { inbox } = await setupSequencerInbox()
    await expect(
      inbox.sendL1FundedUnsignedTransactionToFork(
        0,
        0,
        0,
        ethers.constants.AddressZero,
        '0x'
      )
    ).to.revertedWith('NotForked')
  })

  it('should fail to call sendUnsignedTransactionToFork', async function () {
    const { inbox } = await setupSequencerInbox()
    await expect(
      inbox.sendUnsignedTransactionToFork(
        0,
        0,
        0,
        ethers.constants.AddressZero,
        0,
        '0x'
      )
    ).to.revertedWith('NotForked')
  })

  it('should fail to call sendWithdrawEthToFork', async function () {
    const { inbox } = await setupSequencerInbox()
    await expect(
      inbox.sendWithdrawEthToFork(0, 0, 0, 0, ethers.constants.AddressZero)
    ).to.revertedWith('NotForked')
  })

  it('can upgrade Inbox', async () => {
    const { inboxProxy, inboxTemplate, bridgeProxy } =
      await setupSequencerInbox()

    const currentStorage = []
    for (let i = 0; i < 1024; i++) {
      currentStorage[i] = await inboxProxy.provider!.getStorageAt(
        inboxProxy.address,
        i
      )
    }

    await expect(
      inboxProxy.upgradeToAndCall(
        inboxTemplate.address,
        (
          await inboxTemplate.populateTransaction.postUpgradeInit(
            bridgeProxy.address
          )
        ).data!
      )
    ).to.emit(inboxProxy, 'Upgraded')

    for (let i = 0; i < currentStorage.length; i++) {
      await expect(
        await inboxProxy.provider!.getStorageAt(inboxProxy.address, i)
      ).to.equal(currentStorage[i])
    }
  })
})
