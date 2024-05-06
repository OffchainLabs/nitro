import { ethers } from 'hardhat'
import { ContractFactory, Contract, Overrides } from 'ethers'
import '@nomiclabs/hardhat-ethers'
import { run } from 'hardhat'
import {
  abi as UpgradeExecutorABI,
  bytecode as UpgradeExecutorBytecode,
} from '@offchainlabs/upgrade-executor/build/contracts/src/UpgradeExecutor.sol/UpgradeExecutor.json'
import { maxDataSize } from './config'
import { Toolkit4844 } from '../test/contract/toolkit4844'
import { ArbSys__factory } from '../build/types'
import { ARB_SYS_ADDRESS } from '@arbitrum/sdk/dist/lib/dataEntities/constants'

// Define a verification function
export async function verifyContract(
  contractName: string,
  contractAddress: string,
  constructorArguments: any[] = [],
  contractPathAndName?: string // optional
): Promise<void> {
  try {
    if (process.env.DISABLE_VERIFICATION) return
    // Define the verification options with possible 'contract' property
    const verificationOptions: {
      contract?: string
      address: string
      constructorArguments: any[]
    } = {
      address: contractAddress,
      constructorArguments: constructorArguments,
    }

    // if contractPathAndName is provided, add it to the verification options
    if (contractPathAndName) {
      verificationOptions.contract = contractPathAndName
    }

    await run('verify:verify', verificationOptions)
    console.log(`Verified contract ${contractName} successfully.`)
  } catch (error: any) {
    if (error.message.includes('Already Verified')) {
      console.log(`Contract ${contractName} is already verified.`)
    } else {
      console.error(
        `Verification for ${contractName} failed with the following error: ${error.message}`
      )
    }
  }
}

// Function to handle contract deployment
export async function deployContract(
  contractName: string,
  signer: any,
  constructorArgs: any[] = [],
  verify: boolean = true,
  overrides?: Overrides,
  contractPathAndName?: string // optional
): Promise<Contract> {
  const factory: ContractFactory = await ethers.getContractFactory(contractName)
  const connectedFactory: ContractFactory = factory.connect(signer)

  let deploymentArgs = [...constructorArgs]
  if (overrides) {
    deploymentArgs.push(overrides)
  }

  const contract: Contract = await connectedFactory.deploy(...deploymentArgs)
  await contract.deployTransaction.wait()
  // sleep 3 slots to ensure contract is mined
  await new Promise((r) => setTimeout(r, 3*12000))
  console.log(`New ${contractName} created at address:`, contract.address)

  if (verify)
    await verifyContract(contractName, contract.address, constructorArgs, contractPathAndName)

  return contract
}

// Deploy upgrade executor from imported bytecode
export async function deployUpgradeExecutor(signer: any): Promise<Contract> {
  const upgradeExecutorFac = await ethers.getContractFactory(
    UpgradeExecutorABI,
    UpgradeExecutorBytecode
  )
  const connectedFactory: ContractFactory = upgradeExecutorFac.connect(signer)
  const upgradeExecutor = await connectedFactory.deploy()
  return upgradeExecutor
}

// Function to handle all deployments of core contracts using deployContract function
export async function deployAllContracts(
  signer: any
): Promise<Record<string, Contract>> {
  const isOnArb = await _isRunningOnArbitrum(signer)

  const ethBridge = await deployContract('Bridge', signer, [],true, undefined ,'src/bridge/Bridge.sol:Bridge')
  const reader4844 = isOnArb
    ? ethers.constants.AddressZero
    : (await Toolkit4844.deployReader4844(signer)).address

  const ethSequencerInbox = await deployContract('SequencerInbox', signer, [
    maxDataSize,
    reader4844,
    false,
    false
  ])

  const ethDelayBufferableSequencerInbox = await deployContract('SequencerInbox', signer, [
    maxDataSize,
    reader4844,
    false,
    true
  ])

  const ethInbox = await deployContract('Inbox', signer, [maxDataSize])
  const ethRollupEventInbox = await deployContract(
    'RollupEventInbox',
    signer,
    []
  )
  const ethOutbox = await deployContract('Outbox', signer, [])

  const erc20Bridge = await deployContract('ERC20Bridge', signer, [])
  const erc20SequencerInbox = await deployContract('SequencerInbox', signer, [
    maxDataSize,
    reader4844,
    true,
    false
  ])
  const erc20DelayBufferableSequencerInbox = await deployContract('SequencerInbox', signer, [
    maxDataSize,
    reader4844,
    true,
    true
  ])
  const erc20Inbox = await deployContract('ERC20Inbox', signer, [maxDataSize])
  const erc20RollupEventInbox = await deployContract(
    'ERC20RollupEventInbox',
    signer,
    []
  )
  const erc20Outbox = await deployContract('ERC20Outbox', signer, [])

  const bridgeCreator = await deployContract('BridgeCreator', signer, [
    [
      ethBridge.address,
      ethSequencerInbox.address,
      ethDelayBufferableSequencerInbox.address,
      ethInbox.address,
      ethRollupEventInbox.address,
      ethOutbox.address,
    ],
    [
      erc20Bridge.address,
      erc20SequencerInbox.address,
      erc20DelayBufferableSequencerInbox.address,
      erc20Inbox.address,
      erc20RollupEventInbox.address,
      erc20Outbox.address,
    ]
  ])
  const prover0 = await deployContract('OneStepProver0', signer)
  const proverMem = await deployContract('OneStepProverMemory', signer)
  const proverMath = await deployContract('OneStepProverMath', signer)
  const proverHostIo = await deployContract('OneStepProverHostIo', signer)
  const osp: Contract = await deployContract('OneStepProofEntry', signer, [
    prover0.address,
    proverMem.address,
    proverMath.address,
    proverHostIo.address,
  ])
  const challengeManager = await deployContract('ChallengeManager', signer)
  const rollupAdmin = await deployContract('RollupAdminLogic', signer)
  const rollupUser = await deployContract('RollupUserLogic', signer)
  const upgradeExecutor = await deployUpgradeExecutor(signer)
  const validatorUtils = await deployContract('ValidatorUtils', signer)
  const validatorWalletCreator = await deployContract(
    'ValidatorWalletCreator',
    signer
  )
  const rollupCreator = await deployContract('RollupCreator', signer)
  const deployHelper = await deployContract('DeployHelper', signer)
  return {
    bridgeCreator,
    prover0,
    proverMem,
    proverMath,
    proverHostIo,
    osp,
    challengeManager,
    rollupAdmin,
    rollupUser,
    upgradeExecutor,
    validatorUtils,
    validatorWalletCreator,
    rollupCreator,
    deployHelper,
  }
}

// Check if we're deploying to an Arbitrum chain
async function _isRunningOnArbitrum(signer: any): Promise<Boolean> {
  const arbSys = ArbSys__factory.connect(ARB_SYS_ADDRESS, signer)
  try {
    await arbSys.arbOSVersion()
    return true
  } catch (error) {
    return false
  }
}
