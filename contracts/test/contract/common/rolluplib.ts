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
  RollupLib,
  AssertionCreatedEvent,
} from '../../../build/types/src/rollup/RollupUserLogic.sol/RollupUserLogic'
type AssertionStruct = RollupLib.AssertionStruct
type ExecutionStateStruct = RollupLib.ExecutionStateStruct
import { blockStateHash, hashChallengeState } from './challengeLib'
import * as globalStateLib from './globalStateLib'
import { constants } from 'ethers'
import { GlobalStateStruct } from '../../../build/types/src/challenge/ChallengeManager'

export interface Assertion {
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
  wasmModuleRoot: BytesLike
): BytesLike {
  return ethers.utils.solidityKeccak256(
    ['bool', 'bytes32', 'bytes32', 'bytes32', 'bytes32'],
    [hasSibling, lastHash, assertionExecHash, inboxAcc, wasmModuleRoot]
  )
}

const globalStateEquals = (
  globalState1: GlobalStateStruct,
  globalState2: GlobalStateStruct
) => {
  return (
    globalState1.bytes32Vals[0].toString() ===
      globalState2.bytes32Vals[0].toString() &&
    globalState1.bytes32Vals[1].toString() ===
      globalState2.bytes32Vals[1].toString() &&
    BigNumber.from(globalState1.u64Vals[0]).eq(globalState2.u64Vals[0]) &&
    BigNumber.from(globalState1.u64Vals[1]).eq(globalState2.u64Vals[1])
  )
}

export const executionStateEquals = (
  executionState1: ExecutionStateStruct,
  executionState2: ExecutionStateStruct
) => {
  return (
    globalStateEquals(
      executionState1.globalState,
      executionState2.globalState
    ) &&
    BigNumber.from(executionState1.machineStatus).eq(
      executionState2.machineStatus
    )
  )
}

export const assertionEquals = (
  assertion1: AssertionStruct,
  assertion2: AssertionStruct
) => {
  return (
    executionStateEquals(assertion1.beforeState, assertion2.beforeState) &&
    executionStateEquals(assertion1.afterState, assertion2.afterState) &&
    BigNumber.from(assertion1.numBlocks).eq(assertion2.numBlocks)
  )
}

export function executionStateHash(
  e: ExecutionStateStruct,
  inboxMaxCount: BigNumberish
) {
  return ethers.utils.solidityKeccak256(
    ['bytes32', 'uint256', 'uint8'],
    [globalStateLib.hash(e.globalState), inboxMaxCount, e.machineStatus]
  )
}

export function executionStructHash(e: ExecutionStateStruct) {
  return ethers.utils.solidityKeccak256(
    ['bytes32', 'uint8'],
    [globalStateLib.hash(e.globalState), e.machineStatus]
  )
}

export function assertionExecutionHash(a: AssertionStruct): BytesLike {
  const seg0 = blockStateHash(
    BigNumber.from(a.beforeState.machineStatus),
    globalStateLib.hash(a.beforeState.globalState)
  )
  const seg1 = blockStateHash(
    BigNumber.from(a.afterState.machineStatus),
    globalStateLib.hash(a.afterState.globalState)
  )
  return hashChallengeState(BigNumber.from(0), BigNumber.from(a.numBlocks), [
    seg0,
    seg1,
  ])
}

async function nodeFromAssertionCreatedLog(
  blockNumber: number,
  log: LogDescription
): Promise<Assertion> {
  if (log.name != 'AssertionCreated') {
    throw Error('wrong event type')
  }
  const parsedEv = log.args as AssertionCreatedEvent['args']

  const node: Assertion = {
    assertion: parsedEv.assertion,
    nodeHash: parsedEv.nodeHash,
    wasmModuleRoot: parsedEv.wasmModuleRoot,
    nodeNum: parsedEv.nodeNum.toNumber(),
    proposedBlock: blockNumber,
    inboxMaxCount: parsedEv.inboxMaxCount,
  }
  return node
}

async function nodeFromTx(
  abi: Interface,
  tx: ContractTransaction
): Promise<Assertion> {
  const receipt = await tx.wait()
  if (receipt.logs == undefined) {
    throw Error('expected receipt to have logs')
  }
  const evs = receipt.logs
    .map(log => {
      try {
        return abi.parseLog(log)
      } catch (e) {
        return undefined
      }
    })
    .filter(ev => ev && ev.name == 'AssertionCreated')
  if (evs.length != 1) {
    throw Error('unique event not found')
  }

  return nodeFromAssertionCreatedLog(receipt.blockNumber, evs[0]!)
}

export class RollupContract {
  constructor(public rollup: RollupUserLogic) {}

  connect(signerOrProvider: Signer | Provider | string): RollupContract {
    return new RollupContract(this.rollup.connect(signerOrProvider))
  }

  async stakeOnNewAssertion(
    sequencerInbox: SequencerInbox,
    parentAssertion: {
      nodeHash: BytesLike
      inboxMaxCount: BigNumber
    },
    assertion: AssertionStruct,
    siblingAssertion?: Assertion,
    stakeToAdd?: BigNumber
  ): Promise<{
    tx: ContractTransaction
    node: Assertion
    expectedNewAssertionHash: BytesLike
  }> {
    const inboxPosition = BigNumber.from(
      assertion.afterState.globalState.u64Vals[0]
    ).toNumber()
    const afterInboxAcc =
      inboxPosition > 0
        ? await sequencerInbox.inboxAccs(inboxPosition - 1)
        : constants.HashZero
    const wasmModuleRoot = await this.rollup.wasmModuleRoot()
    const newAssertionHash = nodeHash(
      Boolean(siblingAssertion),
      (siblingAssertion || parentAssertion).nodeHash,
      assertionExecutionHash(assertion),
      afterInboxAcc,
      wasmModuleRoot
    )
    const tx = stakeToAdd
      ? await this.rollup.newStakeOnNewAssertion(
          assertion,
          newAssertionHash,
          parentAssertion.inboxMaxCount,
          {
            value: stakeToAdd,
          }
        )
      : await this.rollup.stakeOnNewAssertion(
          assertion,
          newAssertionHash,
          parentAssertion.inboxMaxCount
        )
    const node = await nodeFromTx(this.rollup.interface, tx)
    return { tx, node, expectedNewAssertionHash: newAssertionHash }
  }

  stakeOnExistingAssertion(
    nodeNum: BigNumberish,
    nodeHash: BytesLike
  ): Promise<ContractTransaction> {
    return this.rollup.stakeOnExistingAssertion(nodeNum, nodeHash)
  }

  confirmNextAssertion(node: Assertion): Promise<ContractTransaction> {
    return this.rollup.confirmNextAssertion(
      node.assertion.afterState.globalState.bytes32Vals[0],
      node.assertion.afterState.globalState.bytes32Vals[1]
    )
  }

  rejectNextAssertion(stakerAddress: string): Promise<ContractTransaction> {
    return this.rollup.rejectNextAssertion(stakerAddress)
  }

  async createChallenge(
    staker1Address: string,
    staker2Address: string,
    node1: Assertion,
    node2: Assertion
  ): Promise<ContractTransaction> {
    return this.rollup.createChallenge(
      [staker1Address, staker2Address],
      [node1.nodeNum, node2.nodeNum],
      [
        node1.assertion.beforeState.machineStatus,
        node1.assertion.afterState.machineStatus,
      ],
      [
        node1.assertion.beforeState.globalState,
        node1.assertion.afterState.globalState,
      ],
      node1.assertion.numBlocks,
      assertionExecutionHash(node2.assertion),
      [node1.proposedBlock, node2.proposedBlock],
      [node1.wasmModuleRoot, node2.wasmModuleRoot]
    )
  }

  addToDeposit(
    staker: string,
    overrides: PayableOverrides = {}
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

  getAssertionStateHash(index: BigNumberish): Promise<string> {
    return this.rollup.getAssertion(index).then(n => n.stateHash)
  }

  latestStakedAssertion(staker: string): Promise<BigNumber> {
    return this.rollup.latestStakedAssertion(staker)
  }

  currentRequiredStake(): Promise<BigNumber> {
    return this.rollup.currentRequiredStake()
  }
}

export async function forceCreateAssertion(
  rollupAdmin: RollupAdminLogic,
  sequencerInbox: SequencerInbox,
  parentAssertion: Assertion,
  assertion: AssertionStruct,
  siblingAssertion?: Assertion
): Promise<{ tx: ContractTransaction; node: Assertion }> {
  const inboxPosition = BigNumber.from(
    assertion.afterState.globalState.u64Vals[0]
  ).toNumber()
  const afterInboxAcc =
    inboxPosition > 0
      ? await sequencerInbox.inboxAccs(inboxPosition - 1)
      : constants.HashZero
  const wasmModuleRoot = await rollupAdmin.wasmModuleRoot()
  const newAssertionHash = nodeHash(
    Boolean(siblingAssertion),
    (siblingAssertion || parentAssertion).nodeHash,
    assertionExecutionHash(assertion),
    afterInboxAcc,
    wasmModuleRoot
  )
  const tx = await rollupAdmin.forceCreateAssertion(
    parentAssertion.nodeNum,
    parentAssertion.inboxMaxCount,
    assertion,
    newAssertionHash
  )
  const node = await nodeFromTx(rollupAdmin.interface, tx)
  return { tx, node }
}
