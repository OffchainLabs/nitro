import { ethers, Wallet } from 'ethers'
import fs from 'fs'
import {
  DeployedContracts,
  getConfig,
  getJsonFile,
} from './common'
import { deployBoldUpgrade } from './boldUpgradeFunctions'
import dotenv from 'dotenv'

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

  const configLocation = process.env.CONFIG_LOCATION
  if (!configLocation) {
    throw new Error('CONFIG_LOCATION env variable not set')
  }
  const config = await getConfig(configLocation, l1Rpc)

  const deployedContractsLocation = process.env.DEPLOYED_CONTRACTS_LOCATION
  if (!deployedContractsLocation) {
    throw new Error('DEPLOYED_CONTRACTS_LOCATION env variable not set')
  }

  // if the deployed contracts exists then we load it and combine
  // if not, then we just use the newly created item
  let existingDeployedContracts = {}
  try {
    existingDeployedContracts = getJsonFile(
      deployedContractsLocation
    ) as DeployedContracts
  } catch (err) {}

  const deployedAndBold = await deployBoldUpgrade(wallet, config, true)

  console.log(`Deployed contracts written to: ${deployedContractsLocation}`)
  fs.writeFileSync(
    deployedContractsLocation,
    JSON.stringify(
      { ...existingDeployedContracts, ...deployedAndBold },
      null,
      2
    )
  )
}

main().then(() => console.log('Done.'))
