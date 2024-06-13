import { ethers, Wallet } from 'ethers'
import { DeployedContracts, getConfig, getJsonFile } from './common'
import { populateLookup } from './boldUpgradeFunctions'
import dotenv from 'dotenv'
import path from 'path'

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

  const configNetworkName = process.env.CONFIG_NETWORK_NAME
  if (!configNetworkName) {
    throw new Error('CONFIG_NETWORK_NAME env variable not set')
  }
  const config = await getConfig(configNetworkName, l1Rpc)

  const deployedContractsDir = process.env.DEPLOYED_CONTRACTS_DIR
  if (!deployedContractsDir) {
    throw new Error('DEPLOYED_CONTRACTS_DIR env variable not set')
  }
  const deployedContractsLocation = path.join(
    deployedContractsDir,
    configNetworkName + 'DeployedContracts.json'
  )
  const deployedContracts = getJsonFile(
    deployedContractsLocation
  ) as DeployedContracts
  if (!deployedContracts?.preImageHashLookup) {
    throw new Error(
      'preImageHashLookup not found in ' + deployedContractsLocation
    )
  }

  await populateLookup(
    wallet,
    config.contracts.rollup,
    deployedContracts.preImageHashLookup,
    deployedContracts.rollupReader
  )
}

// execute this script just prior to execution of the bold upgrade
// it populates the hash lookup contract necessary preimages
main().then(() => console.log('Done.'))
