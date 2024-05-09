import { ethers } from 'hardhat'
import { ContractFactory, Contract, Overrides } from 'ethers'
import '@nomiclabs/hardhat-ethers'
import { IReader4844__factory } from '../../build/types'
import { bytecode as Reader4844Bytecode } from '../../out/yul/Reader4844.yul/Reader4844.json'
import { deployContract, verifyContract } from '../deploymentUtils'
import { maxDataSize, isUsingFeeToken } from '../config'

async function main() {
  const [signer] = await ethers.getSigners()
  const overrides: Overrides = {
    maxFeePerGas: ethers.utils.parseUnits('30', 'gwei'),
    maxPriorityFeePerGas: ethers.utils.parseUnits('0.001', 'gwei'),
  }

  const contractFactory = new ContractFactory(
    IReader4844__factory.abi,
    Reader4844Bytecode,
    signer
  )
  const reader4844 = await contractFactory.deploy(overrides)
  await reader4844.deployed()
  console.log(`Reader4844 deployed at ${reader4844.address}`)

  // skip verification on deployment
  const sequencerInbox = await deployContract(
    'SequencerInbox',
    signer,
    [maxDataSize, reader4844.address, isUsingFeeToken],
    false,
    overrides
  )
  // SequencerInbox logic do not need to be initialized
  const prover0 = await deployContract(
    'OneStepProver0',
    signer,
    [],
    false,
    overrides
  )
  const proverMem = await deployContract(
    'OneStepProverMemory',
    signer,
    [],
    false,
    overrides
  )
  const proverMath = await deployContract(
    'OneStepProverMath',
    signer,
    [],
    false,
    overrides
  )
  const proverHostIo = await deployContract(
    'OneStepProverHostIo',
    signer,
    [],
    false,
    overrides
  )
  const osp: Contract = await deployContract(
    'OneStepProofEntry',
    signer,
    [
      prover0.address,
      proverMem.address,
      proverMath.address,
      proverHostIo.address,
    ],
    false,
    overrides
  )
  const challengeManager = await deployContract(
    'ChallengeManager',
    signer,
    [],
    false,
    overrides
  )
  // ChallengeManager logic do not need to be initialized

  // verify
  await verifyContract('SequencerInbox', sequencerInbox.address, [
    maxDataSize,
    reader4844.address,
    isUsingFeeToken,
  ])
  await verifyContract('OneStepProver0', prover0.address, [])
  await verifyContract('OneStepProverMemory', proverMem.address, [])
  await verifyContract('OneStepProverMath', proverMath.address, [])
  await verifyContract('OneStepProverHostIo', proverHostIo.address, [])
  await verifyContract('OneStepProofEntry', osp.address, [
    prover0.address,
    proverMem.address,
    proverMath.address,
    proverHostIo.address,
  ])
  await verifyContract('ChallengeManager', challengeManager.address, [])
}

main()
  .then(() => process.exit(0))
  .catch((error: Error) => {
    console.error(error)
    process.exit(1)
  })
