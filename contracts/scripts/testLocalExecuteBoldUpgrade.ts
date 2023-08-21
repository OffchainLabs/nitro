import { Contract, ContractReceipt, Wallet, ethers } from 'ethers'
import { DeployedContracts, getConfig, getJsonFile } from './common'
import fs from 'fs'
import { BOLDUpgradeAction__factory } from '../build/types'
import { abi as UpgradeExecutorAbi } from './files/UpgradeExecutor.json'
import dotenv from 'dotenv'
import { RollupMigratedEvent } from '../build/types/src/rollup/BOLDUpgradeAction.sol/BOLDUpgradeAction'

dotenv.config()

async function main() {
  const l1RpcVal = process.env.L1_RPC_URL
  if (!l1RpcVal) {
    throw new Error('L1_RPC_URL env variable not set')
  }
  const l1Rpc = new ethers.providers.JsonRpcProvider(l1RpcVal)

  const l1PrivKey = process.env.L1_PRIV_KEY
  if (!l1PrivKey) {
    throw new Error('L1_PRIV_KEY env variable not set')
  }
  const wallet = new Wallet(l1PrivKey, l1Rpc)

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
  if (!deployedContracts.upgradeExecutor) {
    throw new Error('No upgradeExecutor contract deployed')
  }

  const upExec = new Contract(
    deployedContracts.upgradeExecutor,
    UpgradeExecutorAbi,
    wallet
  )
  const boldAction = BOLDUpgradeAction__factory.connect(
    deployedContracts.boldAction,
    wallet
  )

  // what validators did we have in the old rollup?
  const boldActionPerformData = boldAction.interface.encodeFunctionData(
    'perform',
    [config.validators]
  )

  const receipt = (await (
    await upExec.execute(deployedContracts.boldAction, boldActionPerformData)
  ).wait()) as ContractReceipt

  const parsedLog = boldAction.interface.parseLog(
    receipt.events![receipt.events!.length - 2]
  ).args as RollupMigratedEvent['args']

  console.log(`Deployed contracts written to: ${deployedContractsLocation}`)
  fs.writeFileSync(
    deployedContractsLocation,
    JSON.stringify(
      {
        ...deployedContracts,
        newEdgeChallengeManager: parsedLog.challengeManager,
      },
      null,
      2
    )
  )
}

main().then(() => console.log('Done.'))
