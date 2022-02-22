import {
  GlobalStateStruct,
} from '../../../build/types/RollupUserLogic'
import { solidityKeccak256 } from 'ethers/lib/utils'

export function hash(state: GlobalStateStruct) {
  return solidityKeccak256(
    ['string', 'bytes32', 'bytes32', 'uint64', 'uint64'],
    [
      'Global state:',
      state.bytes32_vals[0],
      state.bytes32_vals[1],
      state.u64_vals[0],
      state.u64_vals[1],
    ],
  )
}
