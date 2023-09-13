import { ethers } from 'ethers'
import { DeployedContracts, getConfig, getJsonFile } from './common'
import dotenv from 'dotenv'
import {
  EdgeChallengeManager__factory,
  RollupUserLogic__factory,
} from '../build/types'

dotenv.config()

async function main() {
  const l1RpcVal = process.env.L1_RPC_URL
  if (!l1RpcVal) {
    throw new Error('L1_RPC_URL env variable not set')
  }
  const l1Rpc = new ethers.providers.JsonRpcProvider(l1RpcVal)

  const configLocation = process.env.CONFIG_LOCATION
  if (!configLocation) {
    throw new Error('CONFIG_LOCATION env variable not set')
  }
  const config = await getConfig(configLocation, l1Rpc)

  const deployedContractsLocation = process.env.DEPLOYED_CONTRACTS_LOCATION
  if (!deployedContractsLocation) {
    throw new Error('DEPLOYED_CONTRACTS_LOCATION env variable not set')
  }

  // load the deployed contracts
  const existingDeployedContracts = getJsonFile(
    deployedContractsLocation
  ) as DeployedContracts

  if (!existingDeployedContracts.newEdgeChallengeManager) {
    throw new Error(
      "newEdgeChallengeManager doesn't exist in deployed contracts"
    )
  }

  const edgeChallengeManager = EdgeChallengeManager__factory.connect(
    existingDeployedContracts.newEdgeChallengeManager,
    l1Rpc
  )

  if ((await edgeChallengeManager.stakeToken()) != config.settings.stakeToken) {
    throw new Error('Stake token address does not match')
  }

  if (
    !(await edgeChallengeManager.stakeAmount()).eq(config.settings.miniStakeAmt)
  ) {
    throw new Error('Mini stake amount does not match')
  }

  if (
    !(await edgeChallengeManager.challengePeriodBlocks()).eq(
      config.settings.confirmPeriodBlocks
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
    !(await edgeChallengeManager.NUM_BIGSTEP_LEVEL()).eq(
      config.settings.numBigStepLevel
    )
  ) {
    throw new Error('Number of big step level does not match')
  }

  const assertionChain = RollupUserLogic__factory.connect(
    await edgeChallengeManager.assertionChain(),
    l1Rpc
  )

  if ((await assertionChain.stakeToken()) != config.settings.stakeToken) {
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
      (await assertionChain.anyTrustFastConfirmer()) !=
      config.settings.anyTrustFastConfirmer
    ) {
      throw new Error('Any trust fast confirmer does not match')
    }
  }
}

main().then(() => console.log('Done.'))
