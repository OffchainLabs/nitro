import { ethers } from 'hardhat'
import { Interface, LogDescription } from '@ethersproject/abi'
import { Signer } from '@ethersproject/abstract-signer'
import { BigNumberish, BigNumber } from '@ethersproject/bignumber'
import { BytesLike } from '@ethersproject/bytes'
import { ContractTransaction, PayableOverrides } from '@ethersproject/contracts'
import { Provider } from '@ethersproject/providers'
import {
  RollupUserLogic,
  RollupAdminLogic,
  SequencerInbox,
} from '../../../build/types'
import {
  AssertionStruct,
  ExecutionStateStruct,
  NodeCreatedEvent,
} from '../../../build/types/RollupUserLogic'
import { blockStateHash, hashChallengeState } from './challengeLib'
import * as globalStateLib from './globalStateLib'
import { constants } from 'ethers'

export interface Node {
  nodeNum: number
  proposedBlock: number
  assertion: AssertionStruct
  inboxMaxCount: BigNumber
  nodeHash: BytesLike
  wasmModuleRoot: BytesLike
}

export function nodeHash(
  hasSibling: boolean,
  lastHash: BytesLike,
  assertionExecHash: BytesLike,
  inboxAcc: BytesLike,
): BytesLike {
  return ethers.utils.solidityKeccak256(
    ['bool', 'bytes32', 'bytes32', 'bytes32'],
    [hasSibling, lastHash, assertionExecHash, inboxAcc],
  )
}

export const assertionEquals = (
  assertion1: AssertionStruct,
  assertion2: AssertionStruct,
) => {
  return (
    assertion1.beforeState.globalState.bytes32Vals[0] ===
      assertion2.beforeState.globalState.bytes32Vals[0] &&
    assertion1.beforeState.globalState.bytes32Vals[1] ===
      assertion2.beforeState.globalState.bytes32Vals[1] &&
    BigNumber.from(assertion1.beforeState.globalState.u64Vals[0]).eq(
      assertion2.beforeState.globalState.u64Vals[0],
    ) &&
    BigNumber.from(assertion1.beforeState.globalState.u64Vals[1]).eq(
      assertion2.beforeState.globalState.u64Vals[1],
    ) &&
    assertion1.afterState.globalState.bytes32Vals[0] ===
      assertion2.afterState.globalState.bytes32Vals[0] &&
    assertion1.afterState.globalState.bytes32Vals[1] ===
      assertion2.afterState.globalState.bytes32Vals[1] &&
    BigNumber.from(assertion1.afterState.globalState.u64Vals[0]).eq(
      assertion2.afterState.globalState.u64Vals[0],
    ) &&
    BigNumber.from(assertion1.afterState.globalState.u64Vals[1]).eq(
      assertion2.afterState.globalState.u64Vals[1],
    ) &&
    BigNumber.from(assertion1.numBlocks).eq(assertion2.numBlocks)
  )
}

export function executionStateHash(
  e: ExecutionStateStruct,
  inboxMaxCount: BigNumberish,
) {
  return ethers.utils.solidityKeccak256(
    ['bytes32', 'uint256', 'uint8'],
    [globalStateLib.hash(e.globalState), inboxMaxCount, e.machineStatus],
  )
}

export function executionStructHash(e: ExecutionStateStruct) {
  return ethers.utils.solidityKeccak256(
    ['bytes32', 'uint8'],
    [globalStateLib.hash(e.globalState), e.machineStatus],
  )
}

export function assertionExecutionHash2(a: AssertionStruct): BytesLike {
  const seg0 = blockStateHash(
    BigNumber.from(a.beforeState.machineStatus),
    globalStateLib.hash(a.beforeState.globalState),
  )
  const seg1 = blockStateHash(
    BigNumber.from(a.afterState.machineStatus),
    globalStateLib.hash(a.afterState.globalState),
  )
  return hashChallengeState(BigNumber.from(0), BigNumber.from(a.numBlocks), [
    seg0,
    seg1,
  ])
}

async function nodeFromNodeCreatedLog(
  blockNumber: number,
  log: LogDescription,
): Promise<{ node: Node }> {
  if (log.name != 'NodeCreated') {
    throw Error('wrong event type')
  }
  const parsedEv = log.args as NodeCreatedEvent['args']

  const node: Node = {
    assertion: parsedEv.assertion,
    nodeHash: parsedEv.nodeHash,
    wasmModuleRoot: parsedEv.wasmModuleRoot,
    nodeNum: parsedEv.nodeNum.toNumber(),
    proposedBlock: blockNumber,
    inboxMaxCount: parsedEv.inboxMaxCount,
  }
  return { node }
}

async function nodeFromTx(
  abi: Interface,
  tx: ContractTransaction,
): Promise<{ node: Node }> {
  const receipt = await tx.wait()
  if (receipt.logs == undefined) {
    throw Error('expected receipt to have logs')
  }
  const evs = receipt.logs
    .map((log) => {
      try {
        return abi.parseLog(log)
      } catch (e) {
        return undefined
      }
    })
    .filter((ev) => ev && ev.name == 'NodeCreated')
  if (evs.length != 1) {
    throw Error('unique event not found')
  }

  return nodeFromNodeCreatedLog(receipt.blockNumber, evs[0]!)
}

export class RollupContract {
  constructor(public rollup: RollupUserLogic) {}

  connect(signerOrProvider: Signer | Provider | string): RollupContract {
    return new RollupContract(this.rollup.connect(signerOrProvider))
  }

  async stakeOnNewNode(
    sequencerInbox: SequencerInbox,
    parentNode: {
      assertion: { afterState: ExecutionStateStruct }
      nodeHash: BytesLike
      inboxMaxCount: BigNumber
    },
    assertion: AssertionStruct,
    siblingNode?: Node,
    stakeToAdd?: BigNumber,
  ): Promise<{
    tx: ContractTransaction
    node: Node
    expectedNewNodeHash: BytesLike
  }> {
    const inboxPosition = BigNumber.from(
      assertion.afterState.globalState.u64Vals[0],
    ).toNumber()
    const afterInboxAcc =
      inboxPosition > 0
        ? await sequencerInbox.inboxAccs(inboxPosition - 1)
        : constants.HashZero
    const newNodeHash = nodeHash(
      !!siblingNode,
      (siblingNode || parentNode).nodeHash,
      assertionExecutionHash2(assertion),
      afterInboxAcc,
    )
    const tx = stakeToAdd
      ? await this.rollup.newStakeOnNewNode(
          assertion,
          newNodeHash,
          parentNode.inboxMaxCount,
          { value: stakeToAdd },
        )
      : await this.rollup.stakeOnNewNode(
          assertion,
          newNodeHash,
          parentNode.inboxMaxCount,
        )
    const { node } = await nodeFromTx(this.rollup.interface, tx)
    return { tx, node, expectedNewNodeHash: newNodeHash }
  }

  stakeOnExistingNode(
    nodeNum: BigNumberish,
    nodeHash: BytesLike,
  ): Promise<ContractTransaction> {
    return this.rollup.stakeOnExistingNode(nodeNum, nodeHash)
  }

  confirmNextNode(
    blockHash: BytesLike,
    sendRoot: BytesLike,
  ): Promise<ContractTransaction> {
    return this.rollup.confirmNextNode(blockHash, sendRoot)
  }

  rejectNextNode(stakerAddress: string): Promise<ContractTransaction> {
    return this.rollup.rejectNextNode(stakerAddress)
  }

  async createChallenge(
    staker1Address: string,
    nodeNum1: BigNumberish,
    staker2Address: string,
    nodeNum2: BigNumberish,
    node1: Node,
    node2: Node,
  ): Promise<ContractTransaction> {
    return this.rollup.createChallenge(
      [staker1Address, staker2Address],
      [nodeNum1, nodeNum2],
      [
        node1.assertion.beforeState.machineStatus,
        node1.assertion.afterState.machineStatus,
      ],
      [
        node1.assertion.beforeState.globalState,
        node1.assertion.afterState.globalState,
      ],
      node1.assertion.numBlocks,
      assertionExecutionHash2(node2.assertion),
      [node1.proposedBlock, node2.proposedBlock],
      [node1.wasmModuleRoot, node2.wasmModuleRoot],
    )
  }

  addToDeposit(
    staker: string,
    overrides: PayableOverrides = {},
  ): Promise<ContractTransaction> {
    return this.rollup.addToDeposit(staker, overrides)
  }

  reduceDeposit(amount: BigNumberish): Promise<ContractTransaction> {
    return this.rollup.reduceDeposit(amount)
  }

  returnOldDeposit(stakerAddress: string): Promise<ContractTransaction> {
    return this.rollup.returnOldDeposit(stakerAddress)
  }

  latestConfirmed(): Promise<BigNumber> {
    return this.rollup.latestConfirmed()
  }

  getNodeStateHash(index: BigNumberish): Promise<string> {
    return this.rollup.getNode(index).then((n) => n.stateHash)
  }

  latestStakedNode(staker: string): Promise<BigNumber> {
    return this.rollup.latestStakedNode(staker)
  }

  currentRequiredStake(): Promise<BigNumber> {
    return this.rollup.currentRequiredStake()
  }
}

export async function forceCreateNode(
  rollupAdmin: RollupAdminLogic,
  sequencerInbox: SequencerInbox,
  parentNode: Node,
  assertion: AssertionStruct,
  siblingNode?: Node,
): Promise<{ tx: ContractTransaction; node: Node }> {
  const inboxPosition = BigNumber.from(
    assertion.afterState.globalState.u64Vals[0],
  ).toNumber()
  const afterInboxAcc =
    inboxPosition > 0
      ? await sequencerInbox.inboxAccs(inboxPosition - 1)
      : constants.HashZero
  const newNodeHash = nodeHash(
    !!siblingNode,
    (siblingNode || parentNode).nodeHash,
    assertionExecutionHash2(assertion),
    afterInboxAcc,
  )
  const tx = await rollupAdmin.forceCreateNode(
    parentNode.nodeNum,
    parentNode.inboxMaxCount,
    assertion,
    newNodeHash,
  )
  const { node } = await nodeFromTx(rollupAdmin.interface, tx)
  return { tx, node }
}
