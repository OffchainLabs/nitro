import { BigNumber } from '@ethersproject/bignumber'
import { solidityKeccak256 } from 'ethers/lib/utils'

export enum MachineStatus {
  RUNNING = 0,
  FINISHED = 1,
  ERRORED = 2,
  TOO_FAR = 3,
}

export function hashChallengeState(
  segmentsStart: BigNumber,
  segmentsLength: BigNumber,
  segments: string[]
) {
  return solidityKeccak256(
    ['uint256', 'uint256', 'bytes32[]'],
    [segmentsStart, segmentsLength, segments]
  )
}

export function blockStateHash(
  machineStatus: BigNumber,
  globalStateHash: string
) {
  const machineStatusNum = machineStatus.toNumber()
  if (machineStatusNum === MachineStatus.FINISHED) {
    return solidityKeccak256(
      ['string', 'bytes32'],
      ['Block state:', globalStateHash]
    )
  } else if (machineStatusNum === MachineStatus.ERRORED) {
    return solidityKeccak256(
      ['string', 'bytes32'],
      ['Block state, errored:', globalStateHash]
    )
  } else if (machineStatusNum === MachineStatus.TOO_FAR) {
    return solidityKeccak256(['string', 'bytes32'], ['Block state, too far:'])
  } else {
    console.log(machineStatus.toNumber())
    throw new Error('BAD_BLOCK_STATUS')
  }
}
