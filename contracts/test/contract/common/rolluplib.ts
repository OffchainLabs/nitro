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
import { machineHash } from './challengeLib'
import * as globalStateLib from './globalStateLib'
import { constants } from 'ethers'
import { GlobalStateStruct } from '../../../build/types/src/challenge/ChallengeManager'

export interface Assertion {
  assertionNum: number
  proposedBlock: number
  assertion: AssertionStruct
  inboxMaxCount: BigNumber
  assertionHash: BytesLike
  wasmModuleRoot: BytesLike
}

export function assertionHash(
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

export function afterStateHash(a: AssertionStruct): BytesLike {
  return ethers.utils.solidityKeccak256(
    ['uint8', 'bytes32'],
    [a.afterState.machineStatus, globalStateLib.hash(a.afterState.globalState)]
  )
}

async function assertionFromAssertionCreatedLog(
  blockNumber: number,
  log: LogDescription
): Promise<Assertion> {
  if (log.name != 'AssertionCreated') {
    throw Error('wrong event type')
  }
  const parsedEv = log.args as AssertionCreatedEvent['args']

  const assertion: Assertion = {
    assertion: parsedEv.assertion,
    assertionHash: parsedEv.assertionHash,
    wasmModuleRoot: parsedEv.wasmModuleRoot,
    assertionNum: parsedEv.assertionNum.toNumber(),
    proposedBlock: blockNumber,
    inboxMaxCount: parsedEv.inboxMaxCount,
  }
  return assertion
}

async function assertionFromTx(
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

  return assertionFromAssertionCreatedLog(receipt.blockNumber, evs[0]!)
}

export class RollupContract {
  constructor(public rollup: RollupUserLogic) {}

  connect(signerOrProvider: Signer | Provider | string): RollupContract {
    return new RollupContract(this.rollup.connect(signerOrProvider))
  }

  async stakeOnNewAssertion(
    sequencerInbox: SequencerInbox,
    parentAssertion: {
      assertionHash: BytesLike
      inboxMaxCount: BigNumber
    },
    assertion: AssertionStruct,
    siblingAssertion?: Assertion,
    stakeToAdd?: BigNumber
  ): Promise<{
    tx: ContractTransaction
    assertion: Assertion
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
    const newAssertionHash = assertionHash(
      Boolean(siblingAssertion),
      (siblingAssertion || parentAssertion).assertionHash,
      afterStateHash(assertion),
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
    const assertion = await assertionFromTx(this.rollup.interface, tx)
    return { tx, assertion, expectedNewAssertionHash: newAssertionHash }
  }

  stakeOnExistingAssertion(
    assertionNum: BigNumberish,
    assertionHash: BytesLike
  ): Promise<ContractTransaction> {
    return this.rollup.stakeOnExistingAssertion(assertionNum, assertionHash)
  }

  confirmNextAssertion(assertion: Assertion): Promise<ContractTransaction> {
    return this.rollup.confirmNextAssertion(
      assertion.assertion.afterState.globalState.bytes32Vals[0],
      assertion.assertion.afterState.globalState.bytes32Vals[1]
    )
  }

  rejectNextAssertion(stakerAddress: string): Promise<ContractTransaction> {
    return this.rollup.rejectNextAssertion(stakerAddress)
  }

  async createChallenge(
    staker1Address: string,
    staker2Address: string,
    assertion1: Assertion,
    assertion2: Assertion
  ): Promise<ContractTransaction> {
    return this.rollup.createChallenge(
      [staker1Address, staker2Address],
      [assertion1.assertionNum, assertion2.assertionNum],
      [
        assertion1.assertion.beforeState.machineStatus,
        assertion1.assertion.afterState.machineStatus,
      ],
      [
        assertion1.assertion.beforeState.globalState,
        assertion1.assertion.afterState.globalState,
      ],
      assertion1.assertion.numBlocks,
      afterStateHash(assertion2.assertion),
      [assertion1.proposedBlock, assertion2.proposedBlock],
      [assertion1.wasmModuleRoot, assertion2.wasmModuleRoot]
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
): Promise<{ tx: ContractTransaction; assertion: Assertion }> {
  const inboxPosition = BigNumber.from(
    assertion.afterState.globalState.u64Vals[0]
  ).toNumber()
  const afterInboxAcc =
    inboxPosition > 0
      ? await sequencerInbox.inboxAccs(inboxPosition - 1)
      : constants.HashZero
  const wasmModuleRoot = await rollupAdmin.wasmModuleRoot()
  const newAssertionHash = assertionHash(
    Boolean(siblingAssertion),
    (siblingAssertion || parentAssertion).assertionHash,
    afterStateHash(assertion),
    afterInboxAcc,
    wasmModuleRoot
  )
  const tx = await rollupAdmin.forceCreateAssertion(
    parentAssertion.assertionNum,
    parentAssertion.inboxMaxCount,
    assertion,
    newAssertionHash
  )
  const assertion = await assertionFromTx(rollupAdmin.interface, tx)
  return { tx, assertion }
}
