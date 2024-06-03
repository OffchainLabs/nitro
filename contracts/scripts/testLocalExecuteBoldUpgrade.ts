import { Contract, ContractReceipt } from 'ethers'
import { ethers } from 'hardhat'
import { Config, DeployedContracts, getConfig, getJsonFile } from './common'
import { BOLDUpgradeAction__factory, EdgeChallengeManager__factory, RollupUserLogic__factory } from '../build/types'
import { abi as UpgradeExecutorAbi } from './files/UpgradeExecutor.json'
import dotenv from 'dotenv'
import { BOLDUpgradeAction, RollupMigratedEvent } from '../build/types/src/rollup/BOLDUpgradeAction.sol/BOLDUpgradeAction'
import { JsonRpcProvider } from '@ethersproject/providers'
import { getAddress } from 'ethers/lib/utils'

dotenv.config()

async function perform(l1Rpc: JsonRpcProvider, config: Config, deployedContracts: DeployedContracts) {
  await l1Rpc.send(
    "hardhat_impersonateAccount",
    ["0xE6841D92B0C345144506576eC13ECf5103aC7f49".toLowerCase()],
  )

  await l1Rpc.send(
    "hardhat_setBalance",
    ["0xE6841D92B0C345144506576eC13ECf5103aC7f49", '0x1000000000000000'],
  )

  const timelockImposter = l1Rpc.getSigner('0xE6841D92B0C345144506576eC13ECf5103aC7f49'.toLowerCase())
  
  const upExec = new Contract(
    config.contracts.upgradeExecutor,
    UpgradeExecutorAbi,
    timelockImposter
  )
  const boldAction = BOLDUpgradeAction__factory.connect(
    deployedContracts.boldAction,
    timelockImposter
  )

  // what validators did we have in the old rollup?
  const boldActionPerformData = boldAction.interface.encodeFunctionData(
    'perform',
    [config.validators]
  )

  return (await (
    await upExec.execute(deployedContracts.boldAction, boldActionPerformData)
  ).wait()) as ContractReceipt
}

async function verifyPostUpgrade(l1Rpc: JsonRpcProvider, config: Config, deployedContracts: DeployedContracts, receipt: ContractReceipt) {
  const boldAction = BOLDUpgradeAction__factory.connect(
    deployedContracts.boldAction,
    l1Rpc
  )

  const parsedLog = boldAction.interface.parseLog(
    receipt.events![receipt.events!.length - 2]
  ).args as RollupMigratedEvent['args']

  const edgeChallengeManager = EdgeChallengeManager__factory.connect(
    parsedLog.challengeManager,
    l1Rpc
  )
  if (getAddress(await edgeChallengeManager.stakeToken()) != getAddress(config.settings.stakeToken)) {
    throw new Error('Stake token address does not match')
  }

  for (let i = 0; i < config.settings.miniStakeAmounts.length; i++) {
    if (
      !(await edgeChallengeManager.stakeAmounts(i)).eq(
        config.settings.miniStakeAmounts[i]
      )
    ) {
      throw new Error('Mini stake amount does not match')
    }
  }

  if (
    !(await edgeChallengeManager.challengePeriodBlocks()).eq(
      config.settings.challengePeriodBlocks
    )
  ) {
    throw new Error('Challenge period blocks does not match')
  }

  if (
    !(await edgeChallengeManager.LAYERZERO_BLOCKEDGE_HEIGHT()).eq(
      config.settings.blockLeafSize
    )
  ) {
    throw new Error('Block leaf size does not match')
  }

  if (
    !(await edgeChallengeManager.LAYERZERO_BIGSTEPEDGE_HEIGHT()).eq(
      config.settings.bigStepLeafSize
    )
  ) {
    throw new Error('Big step leaf size does not match')
  }

  if (
    !(await edgeChallengeManager.LAYERZERO_SMALLSTEPEDGE_HEIGHT()).eq(
      config.settings.smallStepLeafSize
    )
  ) {
    throw new Error('Small step leaf size does not match')
  }

  if (
    (await edgeChallengeManager.NUM_BIGSTEP_LEVEL()) !==
    config.settings.numBigStepLevel
  ) {
    throw new Error('Number of big step level does not match')
  }

  const assertionChain = RollupUserLogic__factory.connect(
    await edgeChallengeManager.assertionChain(),
    l1Rpc
  )

  if (getAddress(await assertionChain.stakeToken()) != getAddress(config.settings.stakeToken)) {
    throw new Error('Stake token address does not match')
  }

  if (
    !(await assertionChain.confirmPeriodBlocks()).eq(
      config.settings.confirmPeriodBlocks
    )
  ) {
    throw new Error('Confirm period blocks does not match')
  }

  if (!(await assertionChain.baseStake()).eq(config.settings.stakeAmt)) {
    throw new Error('Base stake does not match')
  }

  if (config.settings.anyTrustFastConfirmer.length != 0) {
    if (
      getAddress(await assertionChain.anyTrustFastConfirmer()) !==
      getAddress(config.settings.anyTrustFastConfirmer)
    ) {
      throw new Error('Any trust fast confirmer does not match')
    }
  }
}

async function main() {
  const l1RpcVal = process.env.L1_RPC_URL
  if (!l1RpcVal) {
    throw new Error('L1_RPC_URL env variable not set')
  }
  const l1Rpc = new ethers.providers.JsonRpcProvider(l1RpcVal) as JsonRpcProvider

  const deployedContractsLocation = process.env.DEPLOYED_CONTRACTS_LOCATION
  if (!deployedContractsLocation) {
    throw new Error('DEPLOYED_CONTRACTS_LOCATION env variable not set')
  }
  const configLocation = process.env.CONFIG_LOCATION
  if (!configLocation) {
    throw new Error('CONFIG_LOCATION env variable not set')
  }
  const config = await getConfig(configLocation, l1Rpc)

  const deployedContracts = getJsonFile(
    deployedContractsLocation
  ) as DeployedContracts
  if (!deployedContracts.boldAction) {
    throw new Error('No boldAction contract deployed')
  }

  const receipt = await perform(l1Rpc, config, deployedContracts)
  await verifyPostUpgrade(l1Rpc, config, deployedContracts, receipt)
}

main().then(() => console.log('Done.'))
