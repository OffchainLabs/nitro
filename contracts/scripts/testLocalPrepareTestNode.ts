import {
  Contract,
  ContractFactory,
  ContractTransaction,
  Wallet,
  ethers,
} from 'ethers'
import { DeployedContracts, getJsonFile } from './common'
import fs from 'fs'
import path from 'path'
import {
  ProxyAdmin__factory,
  TransparentUpgradeableProxy__factory,
  RollupAdminLogic__factory,
} from '../build/types'
import {
  abi as UpgradeExecutorAbi,
  bytecode as UpgradeExecutorBytecode,
} from './files/UpgradeExecutor.json'
import dotenv from 'dotenv'

dotenv.config()

const transferToUpgradeExec = async (
  rollupAdmin: Wallet,
  rollupAddress: string,
) => {
  const upgradeExecutorImpl = await new ContractFactory(
    UpgradeExecutorAbi,
    UpgradeExecutorBytecode,
    rollupAdmin
  ).deploy()
  await upgradeExecutorImpl.deployed()

  const proxyAdminAddress = "0xa4884de60AEef09b1b35fa255F56ee37198A80B3"

  const upExecProxy = await new TransparentUpgradeableProxy__factory(
    rollupAdmin
  ).deploy(upgradeExecutorImpl.address, proxyAdminAddress, '0x')
  await upExecProxy.deployed()

  const upExec = new Contract(
    upExecProxy.address,
    UpgradeExecutorAbi,
    rollupAdmin
  )
  await (
    (await upExec.functions.initialize(rollupAdmin.address, [
      rollupAdmin.address,
    ])) as ContractTransaction
  ).wait()

  await (
    await RollupAdminLogic__factory.connect(
      rollupAddress,
      rollupAdmin
    ).setOwner(upExec.address)
  ).wait()

  await (
    await ProxyAdmin__factory.connect(
      proxyAdminAddress,
      rollupAdmin
    ).transferOwnership(upExec.address)
  ).wait()

  return upExec
}

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

  const localNetworksPath = path.join(__dirname, './files/localNetwork.json')
  const localNetworks = await getJsonFile(localNetworksPath)
  const rollupAddr = localNetworks['l2Network']['ethBridge']['rollup']
  const upExec = await transferToUpgradeExec(wallet, rollupAddr)

  const deployedContractsLocation = process.env.DEPLOYED_CONTRACTS_LOCATION
  if (!deployedContractsLocation) {
    throw new Error('DEPLOYED_CONTRACTS_LOCATION env variable not set')
  }

  const deployedContracts = getJsonFile(
    deployedContractsLocation
  ) as DeployedContracts
  deployedContracts.upgradeExecutor = upExec.address

  console.log(`Deployed contracts written to: ${deployedContractsLocation}`)
  console.log(JSON.stringify(deployedContracts, null, 2))
  fs.writeFileSync(
    deployedContractsLocation,
    JSON.stringify(deployedContracts, null, 2)
  )
}

main().then(() => console.log('Done.'))
