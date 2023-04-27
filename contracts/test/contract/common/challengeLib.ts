import { BigNumber } from '@ethersproject/bignumber'
import { solidityKeccak256 } from 'ethers/lib/utils'

export enum MachineStatus {
  RUNNING = 0,
  FINISHED = 1,
  ERRORED = 2,
}

export function machineHash(
  machineStatus: BigNumber,
  globalStateHash: string
) {
  const machineStatusNum = machineStatus.toNumber()
  if (machineStatusNum === MachineStatus.FINISHED) {
    return solidityKeccak256(
      ['string', 'bytes32'],
      ['Machine finished:', globalStateHash]
    )
  } else if (machineStatusNum === MachineStatus.ERRORED) {
    return solidityKeccak256(
      ['string', 'bytes32'],
      ['Machine errored:', globalStateHash]
    )
  } else {
    console.log(machineStatus.toNumber())
    throw new Error('BAD_BLOCK_STATUS')
  }
}
