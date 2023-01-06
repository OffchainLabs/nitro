import { GlobalStateStruct } from '../../../build/types/src/rollup/RollupUserLogic.sol/RollupUserLogic'
import { solidityKeccak256 } from 'ethers/lib/utils'

export function hash(state: GlobalStateStruct) {
  return solidityKeccak256(
    ['string', 'bytes32', 'bytes32', 'uint64', 'uint64'],
    [
      'Global state:',
      state.bytes32Vals[0],
      state.bytes32Vals[1],
      state.u64Vals[0],
      state.u64Vals[1],
    ]
  )
}
