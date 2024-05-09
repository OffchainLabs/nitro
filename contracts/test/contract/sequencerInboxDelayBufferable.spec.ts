import { ethers, network } from 'hardhat'
import { Block } from '@ethersproject/providers'
import { data } from './batchData.json'
import { DelayedMsgDelivered } from './types'
import { expect } from 'chai'

import {
  getBatchSpendingReport,
  setupSequencerInbox,
  mineBlocks,
  forceIncludeMessages,
} from './testHelpers'

describe('SequencerInboxDelayBufferable', async () => {
  it('can deplete buffer', async () => {
    const { bridge, sequencerInbox, batchPoster, delayConfig, maxDelay } =
      await setupSequencerInbox(true)
    const delayedInboxPending: DelayedMsgDelivered[] = []
    let delayedMessageCount = await bridge.delayedMessageCount()
    let seqReportedMessageSubCount =
      await bridge.sequencerReportedSubMessageCount()

    expect(delayedMessageCount).to.equal(0)
    expect(seqReportedMessageSubCount).to.equal(0)
    expect(await sequencerInbox.isDelayBufferable()).to.be.true

    let delayBufferData = await sequencerInbox.buffer()

    // full buffers
    expect(delayBufferData.bufferBlocks).to.equal(delayConfig.max)

    await (
      await sequencerInbox
        .connect(batchPoster)
        [
          'addSequencerL2BatchFromOrigin(uint256,bytes,uint256,address,uint256,uint256)'
        ](
          0,
          data,
          delayedMessageCount,
          ethers.constants.AddressZero,
          seqReportedMessageSubCount,
          seqReportedMessageSubCount.add(10),
          { gasLimit: 10000000 }
        )
    )
      .wait()
      .then(res => {
        delayedInboxPending.push(getBatchSpendingReport(res))
      })

    delayedMessageCount = await bridge.delayedMessageCount()
    seqReportedMessageSubCount = await bridge.sequencerReportedSubMessageCount()

    expect(delayedMessageCount).to.equal(1)
    expect(seqReportedMessageSubCount).to.equal(10)
    expect(await sequencerInbox.totalDelayedMessagesRead()).to.equal(0)

    await forceIncludeMessages(
      sequencerInbox,
      delayedInboxPending[0].delayedCount + 1,
      delayedInboxPending[0].delayedMessage,
      'ForceIncludeBlockTooSoon'
    )

    await mineBlocks(7200, 12)

    const txnReciept = await forceIncludeMessages(
      sequencerInbox,
      delayedInboxPending[0].delayedCount + 1,
      delayedInboxPending[0].delayedMessage
    )

    let forceIncludedMsg = delayedInboxPending.shift()
    const delayBlocks =
      txnReciept!.blockNumber -
      forceIncludedMsg!.delayedMessage.header.blockNumber
    const unexpectedDelayBlocks = delayBlocks - delayConfig.threshold.toNumber()
    const replenishAmount = Math.floor(
      (delayBlocks * delayConfig.replenishRateInBasis) / 10000
    )

    expect(await sequencerInbox.totalDelayedMessagesRead()).to.equal(1)

    delayBufferData = await sequencerInbox.buffer()

    // full
    expect(delayBufferData.bufferBlocks).to.equal(delayConfig.max)

    expect(delayBufferData.prevBlockNumber).to.equal(
      forceIncludedMsg?.delayedMessage.header.blockNumber
    )
    expect(delayBufferData.prevSequencedBlockNumber).to.equal(
      txnReciept!.blockNumber
    )

    await (
      await sequencerInbox
        .connect(batchPoster)
        [
          'addSequencerL2BatchFromOrigin(uint256,bytes,uint256,address,uint256,uint256)'
        ](
          2,
          data,
          delayedMessageCount,
          ethers.constants.AddressZero,
          seqReportedMessageSubCount,
          seqReportedMessageSubCount.add(10),
          { gasLimit: 10000000 }
        )
    )
      .wait()
      .then(res => {
        delayedInboxPending.push(getBatchSpendingReport(res))
      })

    await mineBlocks(7200, 12)

    const txnReciept2 = await forceIncludeMessages(
      sequencerInbox,
      delayedInboxPending[0].delayedCount + 1,
      delayedInboxPending[0].delayedMessage
    )
    forceIncludedMsg = delayedInboxPending.shift()
    delayBufferData = await sequencerInbox.buffer()

    const depletedBufferBlocks =
      delayConfig.max - unexpectedDelayBlocks + replenishAmount
    expect(delayBufferData.bufferBlocks).to.equal(depletedBufferBlocks)

    expect(await sequencerInbox.totalDelayedMessagesRead()).to.equal(2)

    expect(delayBufferData.prevBlockNumber).to.equal(
      forceIncludedMsg?.delayedMessage.header.blockNumber
    )
    expect(delayBufferData.prevSequencedBlockNumber).to.equal(
      txnReciept2!.blockNumber
    )

    const deadline = await sequencerInbox.forceInclusionDeadline(
      delayBufferData.prevBlockNumber
    )
    const delayBlocksDeadline =
      depletedBufferBlocks > maxDelay.delayBlocks
        ? maxDelay.delayBlocks
        : depletedBufferBlocks
    expect(deadline).to.equal(
      delayBufferData.prevBlockNumber.add(delayBlocksDeadline)
    )

    const unexpectedDelayBlocks2 = delayBufferData.prevSequencedBlockNumber
      .sub(delayBufferData.prevBlockNumber)
      .sub(delayConfig.threshold)
      .toNumber()
    const delay = delayBufferData.prevSequencedBlockNumber.sub(
      delayBufferData.prevBlockNumber
    )
    const futureBlock =
      forceIncludedMsg!.delayedMessage.header.blockNumber + delay.toNumber()
    const replenishAmount2 = Math.floor(
      (delay.toNumber() * delayConfig.replenishRateInBasis) / 10000
    )
    const deadline2 = await sequencerInbox.forceInclusionDeadline(futureBlock)
    const calcBufferBlocks =
      depletedBufferBlocks - unexpectedDelayBlocks2 >
      delayConfig.threshold.toNumber()
        ? depletedBufferBlocks - unexpectedDelayBlocks2 + replenishAmount2
        : delayConfig.threshold.toNumber()
    const delayBlocksDeadline2 =
      calcBufferBlocks > maxDelay.delayBlocks
        ? maxDelay.delayBlocks
        : calcBufferBlocks
    expect(deadline2).to.equal(futureBlock + delayBlocksDeadline2)
  })

  it('can replenish buffer', async () => {
    const { bridge, sequencerInbox, batchPoster, delayConfig } =
      await setupSequencerInbox(true)
    const delayedInboxPending: DelayedMsgDelivered[] = []
    let delayedMessageCount = await bridge.delayedMessageCount()
    let seqReportedMessageSubCount =
      await bridge.sequencerReportedSubMessageCount()
    let delayBufferData = await sequencerInbox.buffer()
    await (
      await sequencerInbox
        .connect(batchPoster)
        [
          'addSequencerL2BatchFromOrigin(uint256,bytes,uint256,address,uint256,uint256)'
        ](
          0,
          data,
          delayedMessageCount,
          ethers.constants.AddressZero,
          seqReportedMessageSubCount,
          seqReportedMessageSubCount.add(10),
          { gasLimit: 10000000 }
        )
    )
      .wait()
      .then(res => {
        delayedInboxPending.push(getBatchSpendingReport(res))
      })

    delayedMessageCount = await bridge.delayedMessageCount()
    seqReportedMessageSubCount = await bridge.sequencerReportedSubMessageCount()

    await forceIncludeMessages(
      sequencerInbox,
      delayedInboxPending[0].delayedCount + 1,
      delayedInboxPending[0].delayedMessage,
      'ForceIncludeBlockTooSoon'
    )

    await mineBlocks(7200, 12)

    await forceIncludeMessages(
      sequencerInbox,
      delayedInboxPending[0].delayedCount + 1,
      delayedInboxPending[0].delayedMessage
    )

    await (
      await sequencerInbox
        .connect(batchPoster)
        [
          'addSequencerL2BatchFromOrigin(uint256,bytes,uint256,address,uint256,uint256)'
        ](
          2,
          data,
          delayedMessageCount,
          ethers.constants.AddressZero,
          seqReportedMessageSubCount,
          seqReportedMessageSubCount.add(10),
          { gasLimit: 10000000 }
        )
    )
      .wait()
      .then(res => {
        delayedInboxPending.push(getBatchSpendingReport(res))
      })

    const tx = sequencerInbox
      .connect(batchPoster)
      [
        'addSequencerL2BatchFromOrigin(uint256,bytes,uint256,address,uint256,uint256)'
      ](
        3,
        data,
        delayedMessageCount.add(1),
        ethers.constants.AddressZero,
        seqReportedMessageSubCount.add(10),
        seqReportedMessageSubCount.add(20),
        { gasLimit: 10000000 }
      )
    await expect(tx).to.be.revertedWith('DelayProofRequired')

    let nextDelayedMsg = delayedInboxPending.pop()
    await mineBlocks(delayConfig.threshold.toNumber() - 100, 12)

    await (
      await sequencerInbox
        .connect(batchPoster)
        .addSequencerL2BatchFromOriginDelayProof(
          3,
          data,
          delayedMessageCount.add(1),
          ethers.constants.AddressZero,
          seqReportedMessageSubCount.add(10),
          seqReportedMessageSubCount.add(20),
          {
            beforeDelayedAcc: nextDelayedMsg!.delayedAcc,
            delayedMessage: {
              kind: nextDelayedMsg!.delayedMessage.header.kind,
              sender: nextDelayedMsg!.delayedMessage.header.sender,
              blockNumber: nextDelayedMsg!.delayedMessage.header.blockNumber,
              timestamp: nextDelayedMsg!.delayedMessage.header.timestamp,
              inboxSeqNum: nextDelayedMsg!.delayedCount,
              baseFeeL1: nextDelayedMsg!.delayedMessage.header.baseFee,
              messageDataHash:
                nextDelayedMsg!.delayedMessage.header.messageDataHash,
            },
          },
          { gasLimit: 10000000 }
        )
    )
      .wait()
      .then(res => {
        delayedInboxPending.push(getBatchSpendingReport(res))
      })
    delayBufferData = await sequencerInbox.buffer()
    nextDelayedMsg = delayedInboxPending.pop()

    await mineBlocks(delayConfig.threshold.toNumber() - 100, 12)

    await (
      await sequencerInbox
        .connect(batchPoster)
        .addSequencerL2BatchFromOriginDelayProof(
          4,
          data,
          delayedMessageCount.add(2),
          ethers.constants.AddressZero,
          seqReportedMessageSubCount.add(20),
          seqReportedMessageSubCount.add(30),
          {
            beforeDelayedAcc: nextDelayedMsg!.delayedAcc,
            delayedMessage: {
              kind: nextDelayedMsg!.delayedMessage.header.kind,
              sender: nextDelayedMsg!.delayedMessage.header.sender,
              blockNumber: nextDelayedMsg!.delayedMessage.header.blockNumber,
              timestamp: nextDelayedMsg!.delayedMessage.header.timestamp,
              inboxSeqNum: nextDelayedMsg!.delayedCount,
              baseFeeL1: nextDelayedMsg!.delayedMessage.header.baseFee,
              messageDataHash:
                nextDelayedMsg!.delayedMessage.header.messageDataHash,
            },
          },
          { gasLimit: 10000000 }
        )
    )
      .wait()
      .then(res => {
        delayedInboxPending.push(getBatchSpendingReport(res))
        return res
      })

    const delayBufferData2 = await sequencerInbox.buffer()
    const replenishBlocks = Math.floor(
      ((nextDelayedMsg!.delayedMessage.header.blockNumber -
        delayBufferData.prevBlockNumber.toNumber()) *
        delayConfig.replenishRateInBasis) /
        10000
    )
    expect(delayBufferData2.bufferBlocks.toNumber()).to.equal(
      delayBufferData.bufferBlocks.toNumber() + replenishBlocks
    )
  })

  it('happy path', async () => {
    const { bridge, sequencerInbox, batchPoster, delayConfig } =
      await setupSequencerInbox(true)
    const delayedInboxPending: DelayedMsgDelivered[] = []
    const delayedMessageCount = await bridge.delayedMessageCount()
    const seqReportedMessageSubCount =
      await bridge.sequencerReportedSubMessageCount()

    const block = (await network.provider.send('eth_getBlockByNumber', [
      'latest',
      false,
    ])) as Block
    const blockNumber = Number.parseInt(block.number.toString(10))
    expect(
      blockNumber - (await sequencerInbox.buffer()).prevBlockNumber.toNumber()
    ).lessThanOrEqual((await sequencerInbox.buffer()).threshold.toNumber())
    await (
      await sequencerInbox
        .connect(batchPoster)
        [
          'addSequencerL2BatchFromOrigin(uint256,bytes,uint256,address,uint256,uint256)'
        ](
          0,
          data,
          delayedMessageCount,
          ethers.constants.AddressZero,
          seqReportedMessageSubCount,
          seqReportedMessageSubCount.add(10),
          { gasLimit: 10000000 }
        )
    )
      .wait()
      .then(res => {
        delayedInboxPending.push(getBatchSpendingReport(res))
      })

    await mineBlocks(delayConfig.threshold.toNumber() - 100, 12)
    await (
      await sequencerInbox
        .connect(batchPoster)
        .addSequencerL2BatchFromOriginDelayProof(
          1,
          data,
          delayedMessageCount.add(1),
          ethers.constants.AddressZero,
          seqReportedMessageSubCount.add(10),
          seqReportedMessageSubCount.add(20),
          {
            beforeDelayedAcc: delayedInboxPending[0].delayedAcc,
            delayedMessage: {
              kind: delayedInboxPending[0].delayedMessage.header.kind,
              sender: delayedInboxPending[0].delayedMessage.header.sender,
              blockNumber:
                delayedInboxPending[0].delayedMessage.header.blockNumber,
              timestamp: delayedInboxPending[0].delayedMessage.header.timestamp,
              inboxSeqNum: delayedInboxPending[0].delayedCount,
              baseFeeL1: delayedInboxPending[0].delayedMessage.header.baseFee,
              messageDataHash:
                delayedInboxPending[0].delayedMessage.header.messageDataHash,
            },
          },
          { gasLimit: 10000000 }
        )
    ).wait()

    // sequencerReportedSubMessageCount
    expect(await bridge.sequencerReportedSubMessageCount()).to.equal(20)
    //seqMessageIndex
    expect(await bridge.sequencerMessageCount()).to.equal(2)
  })

  it('unhappy path', async () => {
    const { bridge, sequencerInbox, batchPoster, delayConfig } =
      await setupSequencerInbox(true)
    let delayedInboxPending: DelayedMsgDelivered[] = []
    const delayedMessageCount = await bridge.delayedMessageCount()
    const seqReportedMessageSubCount =
      await bridge.sequencerReportedSubMessageCount()

    const block = (await network.provider.send('eth_getBlockByNumber', [
      'latest',
      false,
    ])) as Block
    const blockNumber = Number.parseInt(block.number.toString(10))
    expect(
      blockNumber - (await sequencerInbox.buffer()).prevBlockNumber.toNumber()
    ).lessThanOrEqual((await sequencerInbox.buffer()).threshold.toNumber())
    await (
      await sequencerInbox
        .connect(batchPoster)
        [
          'addSequencerL2BatchFromOrigin(uint256,bytes,uint256,address,uint256,uint256)'
        ](
          0,
          data,
          delayedMessageCount,
          ethers.constants.AddressZero,
          seqReportedMessageSubCount,
          seqReportedMessageSubCount.add(10),
          { gasLimit: 10000000 }
        )
    )
      .wait()
      .then(res => {
        delayedInboxPending.push(getBatchSpendingReport(res))
      })

    await mineBlocks(delayConfig.threshold.toNumber() - 100, 12)
    await (
      await sequencerInbox
        .connect(batchPoster)
        [
          'addSequencerL2BatchFromOrigin(uint256,bytes,uint256,address,uint256,uint256)'
        ](
          1,
          data,
          delayedMessageCount,
          ethers.constants.AddressZero,
          seqReportedMessageSubCount.add(10),
          seqReportedMessageSubCount.add(20),
          { gasLimit: 10000000 }
        )
    )
      .wait()
      .then(res => {
        delayedInboxPending.push(getBatchSpendingReport(res))
      })

    let firstReadMsg = delayedInboxPending[0]
    await mineBlocks(101, 12)

    const txn = sequencerInbox
      .connect(batchPoster)
      [
        'addSequencerL2BatchFromOrigin(uint256,bytes,uint256,address,uint256,uint256)'
      ](
        2,
        data,
        delayedMessageCount.add(2),
        ethers.constants.AddressZero,
        seqReportedMessageSubCount.add(20),
        seqReportedMessageSubCount.add(30),
        { gasLimit: 10000000 }
      )
    await expect(txn).to.be.revertedWith('DelayProofRequired')

    await (
      await sequencerInbox
        .connect(batchPoster)
        .addSequencerL2BatchFromOriginDelayProof(
          2,
          data,
          delayedMessageCount.add(2),
          ethers.constants.AddressZero,
          seqReportedMessageSubCount.add(20),
          seqReportedMessageSubCount.add(30),
          {
            beforeDelayedAcc: firstReadMsg!.delayedAcc,
            delayedMessage: {
              kind: firstReadMsg!.delayedMessage.header.kind,
              sender: firstReadMsg!.delayedMessage.header.sender,
              blockNumber: firstReadMsg!.delayedMessage.header.blockNumber,
              timestamp: firstReadMsg!.delayedMessage.header.timestamp,
              inboxSeqNum: firstReadMsg!.delayedCount,
              baseFeeL1: firstReadMsg!.delayedMessage.header.baseFee,
              messageDataHash:
                firstReadMsg!.delayedMessage.header.messageDataHash,
            },
          },
          { gasLimit: 10000000 }
        )
    )
      .wait()
      .then(async res => {
        delayedInboxPending = []
        delayedInboxPending.push(getBatchSpendingReport(res))
        return res
      })

    const delayBufferDataBefore = await sequencerInbox.buffer()
    firstReadMsg = delayedInboxPending[0]
    await (
      await sequencerInbox
        .connect(batchPoster)
        .addSequencerL2BatchFromOriginDelayProof(
          3,
          data,
          delayedMessageCount.add(3),
          ethers.constants.AddressZero,
          seqReportedMessageSubCount.add(30),
          seqReportedMessageSubCount.add(40),
          {
            beforeDelayedAcc: firstReadMsg!.delayedAcc,
            delayedMessage: {
              kind: firstReadMsg!.delayedMessage.header.kind,
              sender: firstReadMsg!.delayedMessage.header.sender,
              blockNumber: firstReadMsg!.delayedMessage.header.blockNumber,
              timestamp: firstReadMsg!.delayedMessage.header.timestamp,
              inboxSeqNum: firstReadMsg!.delayedCount,
              baseFeeL1: firstReadMsg!.delayedMessage.header.baseFee,
              messageDataHash:
                firstReadMsg!.delayedMessage.header.messageDataHash,
            },
          },
          { gasLimit: 10000000 }
        )
    )
      .wait()
      .then(async res => {
        delayedInboxPending = []
        delayedInboxPending.push(getBatchSpendingReport(res))
        return res
      })

    const unexpectedDelayBlocks =
      delayBufferDataBefore.prevSequencedBlockNumber.toNumber() -
      delayBufferDataBefore.prevBlockNumber.toNumber() -
      delayConfig.threshold.toNumber()
    const elapsed =
      firstReadMsg!.delayedMessage.header.blockNumber -
      delayBufferDataBefore.prevBlockNumber.toNumber()
    const replenishAmount = Math.floor(
      (elapsed * delayConfig.replenishRateInBasis) / 10000
    )
    const bufferBlocksUpdate =
      delayBufferDataBefore.bufferBlocks.toNumber() -
      Math.min(unexpectedDelayBlocks, elapsed) +
      replenishAmount
    expect((await sequencerInbox.buffer()).bufferBlocks).to.equal(
      Math.min(bufferBlocksUpdate, delayConfig.max)
    )
  })
})

describe('SequencerInboxDelayBufferableBlobMock', async () => {
  it('can deplete buffer', async () => {
    const { bridge, sequencerInbox, batchPoster, delayConfig, maxDelay } =
      await setupSequencerInbox(true, true)
    const delayedInboxPending: DelayedMsgDelivered[] = []
    let delayedMessageCount = await bridge.delayedMessageCount()
    let seqReportedMessageSubCount =
      await bridge.sequencerReportedSubMessageCount()

    expect(delayedMessageCount).to.equal(0)
    expect(seqReportedMessageSubCount).to.equal(0)
    expect(await sequencerInbox.isDelayBufferable()).to.be.true

    let delayBufferData = await sequencerInbox.buffer()

    // full buffer
    expect(delayBufferData.bufferBlocks).to.equal(delayConfig.max)

    await (
      await sequencerInbox
        .connect(batchPoster)
        .addSequencerL2BatchFromBlobs(
          0,
          delayedMessageCount,
          ethers.constants.AddressZero,
          seqReportedMessageSubCount,
          seqReportedMessageSubCount.add(10),
          { gasLimit: 10000000 }
        )
    )
      .wait()
      .then(res => {
        delayedInboxPending.push(getBatchSpendingReport(res))
      })

    delayedMessageCount = await bridge.delayedMessageCount()
    seqReportedMessageSubCount = await bridge.sequencerReportedSubMessageCount()

    expect(delayedMessageCount).to.equal(1)
    expect(seqReportedMessageSubCount).to.equal(10)
    expect(await sequencerInbox.totalDelayedMessagesRead()).to.equal(0)

    await forceIncludeMessages(
      sequencerInbox,
      delayedInboxPending[0].delayedCount + 1,
      delayedInboxPending[0].delayedMessage,
      'ForceIncludeBlockTooSoon'
    )

    await mineBlocks(7200, 12)

    const txnReciept = await forceIncludeMessages(
      sequencerInbox,
      delayedInboxPending[0].delayedCount + 1,
      delayedInboxPending[0].delayedMessage
    )

    let forceIncludedMsg = delayedInboxPending.pop()
    const delayBlocks =
      txnReciept!.blockNumber -
      forceIncludedMsg!.delayedMessage.header.blockNumber
    const unexpectedDelayBlocks = delayBlocks - delayConfig.threshold.toNumber()
    const replenishAmount = Math.floor(
      (delayBlocks * delayConfig.replenishRateInBasis) / 10000
    )

    expect(await sequencerInbox.totalDelayedMessagesRead()).to.equal(1)

    delayBufferData = await sequencerInbox.buffer()

    // full
    expect(delayBufferData.bufferBlocks).to.equal(delayConfig.max)

    expect(delayBufferData.prevBlockNumber).to.equal(
      forceIncludedMsg?.delayedMessage.header.blockNumber
    )

    expect(delayBufferData.prevSequencedBlockNumber).to.equal(
      txnReciept!.blockNumber
    )

    await (
      await sequencerInbox
        .connect(batchPoster)
        .addSequencerL2BatchFromBlobs(
          2,
          delayedMessageCount,
          ethers.constants.AddressZero,
          seqReportedMessageSubCount,
          seqReportedMessageSubCount.add(10),
          { gasLimit: 10000000 }
        )
    )
      .wait()
      .then(res => {
        delayedInboxPending.push(getBatchSpendingReport(res))
      })

    await mineBlocks(7200, 12)

    const txnReciept2 = await forceIncludeMessages(
      sequencerInbox,
      delayedInboxPending[0].delayedCount + 1,
      delayedInboxPending[0].delayedMessage
    )
    forceIncludedMsg = delayedInboxPending.pop()
    delayBufferData = await sequencerInbox.buffer()

    const depletedBufferBlocks =
      delayConfig.max - unexpectedDelayBlocks + replenishAmount
    expect(delayBufferData.bufferBlocks).to.equal(depletedBufferBlocks)

    expect(await sequencerInbox.totalDelayedMessagesRead()).to.equal(2)

    expect(delayBufferData.prevBlockNumber).to.equal(
      forceIncludedMsg?.delayedMessage.header.blockNumber
    )
    expect(delayBufferData.prevSequencedBlockNumber).to.equal(
      txnReciept2!.blockNumber
    )

    const deadline = await sequencerInbox.forceInclusionDeadline(
      delayBufferData.prevBlockNumber
    )
    const delayBlocksDeadline =
      depletedBufferBlocks > maxDelay.delayBlocks
        ? maxDelay.delayBlocks
        : depletedBufferBlocks
    expect(deadline).to.equal(
      delayBufferData.prevBlockNumber.add(delayBlocksDeadline)
    )
    const delay = delayBufferData.prevSequencedBlockNumber.sub(
      delayBufferData.prevBlockNumber
    )
    const unexpectedDelayBlocks2 = delay.sub(delayConfig.threshold).toNumber()
    const futureBlock =
      forceIncludedMsg!.delayedMessage.header.blockNumber + delay.toNumber()
    const deadline2 = await sequencerInbox.forceInclusionDeadline(futureBlock)
    const replenishAmount2 = Math.floor(
      (delay.toNumber() * delayConfig.replenishRateInBasis) / 10000
    )
    const calcBufferBlocks =
      depletedBufferBlocks - unexpectedDelayBlocks2 >
      delayConfig.threshold.toNumber()
        ? depletedBufferBlocks - unexpectedDelayBlocks2 + replenishAmount2
        : delayConfig.threshold.toNumber()
    const delayBlocksDeadline2 =
      calcBufferBlocks > maxDelay.delayBlocks
        ? maxDelay.delayBlocks
        : calcBufferBlocks
    expect(deadline2).to.equal(futureBlock + delayBlocksDeadline2)
  })

  it('can replenish buffer', async () => {
    const { bridge, sequencerInbox, batchPoster, delayConfig } =
      await setupSequencerInbox(true, true)
    const delayedInboxPending: DelayedMsgDelivered[] = []
    let delayedMessageCount = await bridge.delayedMessageCount()
    let seqReportedMessageSubCount =
      await bridge.sequencerReportedSubMessageCount()
    let delayBufferData = await sequencerInbox.buffer()
    await (
      await sequencerInbox
        .connect(batchPoster)
        .addSequencerL2BatchFromBlobs(
          0,
          delayedMessageCount,
          ethers.constants.AddressZero,
          seqReportedMessageSubCount,
          seqReportedMessageSubCount.add(10),
          { gasLimit: 10000000 }
        )
    )
      .wait()
      .then(res => {
        delayedInboxPending.push(getBatchSpendingReport(res))
      })

    delayedMessageCount = await bridge.delayedMessageCount()
    seqReportedMessageSubCount = await bridge.sequencerReportedSubMessageCount()

    await forceIncludeMessages(
      sequencerInbox,
      delayedInboxPending[0].delayedCount + 1,
      delayedInboxPending[0].delayedMessage,
      'ForceIncludeBlockTooSoon'
    )

    await mineBlocks(7200, 12)

    await forceIncludeMessages(
      sequencerInbox,
      delayedInboxPending[0].delayedCount + 1,
      delayedInboxPending[0].delayedMessage
    )

    await (
      await sequencerInbox
        .connect(batchPoster)
        .addSequencerL2BatchFromBlobs(
          2,
          delayedMessageCount,
          ethers.constants.AddressZero,
          seqReportedMessageSubCount,
          seqReportedMessageSubCount.add(10),
          { gasLimit: 10000000 }
        )
    )
      .wait()
      .then(res => {
        delayedInboxPending.push(getBatchSpendingReport(res))
      })

    const tx = sequencerInbox
      .connect(batchPoster)
      .addSequencerL2BatchFromBlobs(
        3,
        delayedMessageCount.add(1),
        ethers.constants.AddressZero,
        seqReportedMessageSubCount.add(10),
        seqReportedMessageSubCount.add(20),
        { gasLimit: 10000000 }
      )
    await expect(tx).to.be.revertedWith('DelayProofRequired')

    let nextDelayedMsg = delayedInboxPending.pop()
    await mineBlocks(delayConfig.threshold.toNumber() - 100, 12)

    await (
      await sequencerInbox
        .connect(batchPoster)
        .addSequencerL2BatchFromBlobsDelayProof(
          3,
          delayedMessageCount.add(1),
          ethers.constants.AddressZero,
          seqReportedMessageSubCount.add(10),
          seqReportedMessageSubCount.add(20),
          {
            beforeDelayedAcc: nextDelayedMsg!.delayedAcc,
            delayedMessage: {
              kind: nextDelayedMsg!.delayedMessage.header.kind,
              sender: nextDelayedMsg!.delayedMessage.header.sender,
              blockNumber: nextDelayedMsg!.delayedMessage.header.blockNumber,
              timestamp: nextDelayedMsg!.delayedMessage.header.timestamp,
              inboxSeqNum: nextDelayedMsg!.delayedCount,
              baseFeeL1: nextDelayedMsg!.delayedMessage.header.baseFee,
              messageDataHash:
                nextDelayedMsg!.delayedMessage.header.messageDataHash,
            },
          },
          { gasLimit: 10000000 }
        )
    )
      .wait()
      .then(res => {
        delayedInboxPending.push(getBatchSpendingReport(res))
      })
    delayBufferData = await sequencerInbox.buffer()
    nextDelayedMsg = delayedInboxPending.pop()

    await mineBlocks(delayConfig.threshold.toNumber() - 100, 12)

    await (
      await sequencerInbox
        .connect(batchPoster)
        .connect(batchPoster)
        .addSequencerL2BatchFromBlobsDelayProof(
          4,
          delayedMessageCount.add(2),
          ethers.constants.AddressZero,
          seqReportedMessageSubCount.add(20),
          seqReportedMessageSubCount.add(30),
          {
            beforeDelayedAcc: nextDelayedMsg!.delayedAcc,
            delayedMessage: {
              kind: nextDelayedMsg!.delayedMessage.header.kind,
              sender: nextDelayedMsg!.delayedMessage.header.sender,
              blockNumber: nextDelayedMsg!.delayedMessage.header.blockNumber,
              timestamp: nextDelayedMsg!.delayedMessage.header.timestamp,
              inboxSeqNum: nextDelayedMsg!.delayedCount,
              baseFeeL1: nextDelayedMsg!.delayedMessage.header.baseFee,
              messageDataHash:
                nextDelayedMsg!.delayedMessage.header.messageDataHash,
            },
          },
          { gasLimit: 10000000 }
        )
    )
      .wait()
      .then(res => {
        delayedInboxPending.push(getBatchSpendingReport(res))
        return res
      })

    const delayBufferData2 = await sequencerInbox.buffer()
    const replenishBlocks = Math.floor(
      ((nextDelayedMsg!.delayedMessage.header.blockNumber -
        delayBufferData.prevBlockNumber.toNumber()) *
        delayConfig.replenishRateInBasis) /
        10000
    )
    expect(delayBufferData2.bufferBlocks.toNumber()).to.equal(
      delayBufferData.bufferBlocks.toNumber() + replenishBlocks
    )
  })

  it('happy path', async () => {
    const { bridge, sequencerInbox, batchPoster, delayConfig } =
      await setupSequencerInbox(true, true)
    const delayedInboxPending: DelayedMsgDelivered[] = []
    const delayedMessageCount = await bridge.delayedMessageCount()
    const seqReportedMessageSubCount =
      await bridge.sequencerReportedSubMessageCount()

    const block = (await network.provider.send('eth_getBlockByNumber', [
      'latest',
      false,
    ])) as Block
    const blockNumber = Number.parseInt(block.number.toString(10))
    expect(
      blockNumber - (await sequencerInbox.buffer()).prevBlockNumber.toNumber()
    ).lessThanOrEqual((await sequencerInbox.buffer()).threshold.toNumber())
    await (
      await sequencerInbox
        .connect(batchPoster)
        .addSequencerL2BatchFromBlobs(
          0,
          delayedMessageCount,
          ethers.constants.AddressZero,
          seqReportedMessageSubCount,
          seqReportedMessageSubCount.add(10),
          { gasLimit: 10000000 }
        )
    )
      .wait()
      .then(res => {
        delayedInboxPending.push(getBatchSpendingReport(res))
      })

    await mineBlocks(delayConfig.threshold.toNumber() - 100, 12)
    await (
      await sequencerInbox
        .connect(batchPoster)
        .addSequencerL2BatchFromBlobs(
          1,
          delayedMessageCount.add(1),
          ethers.constants.AddressZero,
          seqReportedMessageSubCount.add(10),
          seqReportedMessageSubCount.add(20),
          { gasLimit: 10000000 }
        )
    )
      .wait()
      .then(res => {
        delayedInboxPending.push(getBatchSpendingReport(res))
        return res
      })
  })

  it('unhappy path', async () => {
    const { bridge, sequencerInbox, batchPoster, delayConfig } =
      await setupSequencerInbox(true, true)
    let delayedInboxPending: DelayedMsgDelivered[] = []
    const delayedMessageCount = await bridge.delayedMessageCount()
    const seqReportedMessageSubCount =
      await bridge.sequencerReportedSubMessageCount()

    const block = (await network.provider.send('eth_getBlockByNumber', [
      'latest',
      false,
    ])) as Block
    const blockNumber = Number.parseInt(block.number.toString(10))
    expect(
      blockNumber - (await sequencerInbox.buffer()).prevBlockNumber.toNumber()
    ).lessThanOrEqual((await sequencerInbox.buffer()).threshold.toNumber())
    await (
      await sequencerInbox
        .connect(batchPoster)
        .addSequencerL2BatchFromBlobs(
          0,
          delayedMessageCount,
          ethers.constants.AddressZero,
          seqReportedMessageSubCount,
          seqReportedMessageSubCount.add(10),
          { gasLimit: 10000000 }
        )
    )
      .wait()
      .then(res => {
        delayedInboxPending.push(getBatchSpendingReport(res))
      })

    await mineBlocks(delayConfig.threshold.toNumber() - 100, 12)
    await (
      await sequencerInbox
        .connect(batchPoster)
        .addSequencerL2BatchFromBlobs(
          1,
          delayedMessageCount,
          ethers.constants.AddressZero,
          seqReportedMessageSubCount.add(10),
          seqReportedMessageSubCount.add(20),
          { gasLimit: 10000000 }
        )
    )
      .wait()
      .then(res => {
        delayedInboxPending.push(getBatchSpendingReport(res))
      })

    let firstReadMsg = delayedInboxPending[0]
    await mineBlocks(101, 12)

    const txn = sequencerInbox
      .connect(batchPoster)
      .addSequencerL2BatchFromBlobs(
        2,
        delayedMessageCount.add(2),
        ethers.constants.AddressZero,
        seqReportedMessageSubCount.add(20),
        seqReportedMessageSubCount.add(30),
        { gasLimit: 10000000 }
      )
    await expect(txn).to.be.revertedWith('DelayProofRequired')

    await (
      await sequencerInbox
        .connect(batchPoster)
        .addSequencerL2BatchFromBlobsDelayProof(
          2,
          delayedMessageCount.add(2),
          ethers.constants.AddressZero,
          seqReportedMessageSubCount.add(20),
          seqReportedMessageSubCount.add(30),
          {
            beforeDelayedAcc: firstReadMsg!.delayedAcc,
            delayedMessage: {
              kind: firstReadMsg!.delayedMessage.header.kind,
              sender: firstReadMsg!.delayedMessage.header.sender,
              blockNumber: firstReadMsg!.delayedMessage.header.blockNumber,
              timestamp: firstReadMsg!.delayedMessage.header.timestamp,
              inboxSeqNum: firstReadMsg!.delayedCount,
              baseFeeL1: firstReadMsg!.delayedMessage.header.baseFee,
              messageDataHash:
                firstReadMsg!.delayedMessage.header.messageDataHash,
            },
          },
          { gasLimit: 10000000 }
        )
    )
      .wait()
      .then(async res => {
        delayedInboxPending = []
        delayedInboxPending.push(getBatchSpendingReport(res))
        return res
      })

    const delayBufferDataBefore = await sequencerInbox.buffer()
    firstReadMsg = delayedInboxPending[0]
    await (
      await sequencerInbox
        .connect(batchPoster)
        .addSequencerL2BatchFromBlobsDelayProof(
          3,
          delayedMessageCount.add(3),
          ethers.constants.AddressZero,
          seqReportedMessageSubCount.add(30),
          seqReportedMessageSubCount.add(40),
          {
            beforeDelayedAcc: firstReadMsg!.delayedAcc,
            delayedMessage: {
              kind: firstReadMsg!.delayedMessage.header.kind,
              sender: firstReadMsg!.delayedMessage.header.sender,
              blockNumber: firstReadMsg!.delayedMessage.header.blockNumber,
              timestamp: firstReadMsg!.delayedMessage.header.timestamp,
              inboxSeqNum: firstReadMsg!.delayedCount,
              baseFeeL1: firstReadMsg!.delayedMessage.header.baseFee,
              messageDataHash:
                firstReadMsg!.delayedMessage.header.messageDataHash,
            },
          },
          { gasLimit: 10000000 }
        )
    )
      .wait()
      .then(async res => {
        delayedInboxPending = []
        delayedInboxPending.push(getBatchSpendingReport(res))
        return res
      })

    const unexpectedDelayBlocks =
      delayBufferDataBefore.prevSequencedBlockNumber.toNumber() -
      delayBufferDataBefore.prevBlockNumber.toNumber() -
      delayConfig.threshold.toNumber()
    const elapsed =
      firstReadMsg!.delayedMessage.header.blockNumber -
      delayBufferDataBefore.prevBlockNumber.toNumber()
    const replenishAmount = Math.floor(
      (elapsed * delayConfig.replenishRateInBasis) / 10000
    )
    const bufferBlocksUpdate =
      delayBufferDataBefore.bufferBlocks.toNumber() -
      Math.min(unexpectedDelayBlocks, elapsed) +
      replenishAmount
    expect((await sequencerInbox.buffer()).bufferBlocks).to.equal(
      Math.min(bufferBlocksUpdate, delayConfig.max)
    )
  })
})
