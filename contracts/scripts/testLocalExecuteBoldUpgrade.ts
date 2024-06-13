import { Contract, ContractReceipt } from 'ethers'
import { ethers } from 'hardhat'
import { Config, DeployedContracts, getConfig, getJsonFile } from './common'
import {
  BOLDUpgradeAction__factory,
  Bridge,
  Bridge__factory,
  EdgeChallengeManager,
  EdgeChallengeManager__factory,
  Outbox__factory,
  RollupEventInbox__factory,
  RollupUserLogic,
  RollupUserLogic__factory,
  SequencerInbox__factory,
} from '../build/types'
import { abi as UpgradeExecutorAbi } from './files/UpgradeExecutor.json'
import dotenv from 'dotenv'
import { RollupMigratedEvent } from '../build/types/src/rollup/BOLDUpgradeAction.sol/BOLDUpgradeAction'
import { abi as OldRollupAbi } from './files/OldRollupUserLogic.json'
import { JsonRpcProvider } from '@ethersproject/providers'
import { getAddress } from 'ethers/lib/utils'
import path from 'path'

dotenv.config()

type UnwrapPromise<T> = T extends Promise<infer U> ? U : T

type VerificationParams = {
  l1Rpc: JsonRpcProvider
  config: Config
  deployedContracts: DeployedContracts
  preUpgradeState: UnwrapPromise<ReturnType<typeof getPreUpgradeState>>
  receipt: ContractReceipt
}

const executors: {[key: string]: string} = {
  // DAO L1 Timelocks
  arb1: '0xE6841D92B0C345144506576eC13ECf5103aC7f49',
  nova: '0xE6841D92B0C345144506576eC13ECf5103aC7f49',
  sepolia: '0x6EC62D826aDc24AeA360be9cF2647c42b9Cdb19b'
}

async function getPreUpgradeState(l1Rpc: JsonRpcProvider, config: Config) {
  const oldRollupContract = new Contract(
    config.contracts.rollup,
    OldRollupAbi,
    l1Rpc
  )

  const stakerCount = await oldRollupContract.stakerCount()

  const stakers: string[] = []
  for (let i = 0; i < stakerCount; i++) {
    stakers.push(await oldRollupContract.getStakerAddress(i))
  }

  const boxes = await getAllowedInboxesOutboxesFromBridge(
    Bridge__factory.connect(config.contracts.bridge, l1Rpc)
  )

  const wasmModuleRoot = await oldRollupContract.wasmModuleRoot()

  return {
    stakers,
    wasmModuleRoot,
    ...boxes,
  }
}

async function perform(
  l1Rpc: JsonRpcProvider,
  config: Config,
  deployedContracts: DeployedContracts
) {
  const executor = executors[process.env.CONFIG_NETWORK_NAME!]
  if (!executor) {
    throw new Error('no executor found for CONFIG_NETWORK_NAME or CONFIG_NETWORK_NAME not set')
  }
  await l1Rpc.send('hardhat_impersonateAccount', [executor])

  await l1Rpc.send('hardhat_setBalance', [executor, '0x1000000000000000'])

  const timelockImposter = l1Rpc.getSigner(executor)

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

async function verifyPostUpgrade(params: VerificationParams) {
  const { l1Rpc, deployedContracts, receipt } = params

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

  const newRollup = RollupUserLogic__factory.connect(parsedLog.rollup, l1Rpc)

  await checkSequencerInbox(params, newRollup)
  await checkInbox(params)
  await checkBridge(params, newRollup)
  await checkRollupEventInbox(params, newRollup)
  await checkOutbox(params, newRollup)
  await checkOldRollup(params)
  await checkNewRollup(params, newRollup, edgeChallengeManager)
  await checkNewChallengeManager(params, newRollup, edgeChallengeManager)
}

async function checkSequencerInbox(
  params: VerificationParams,
  newRollup: RollupUserLogic
) {
  const { l1Rpc, config, deployedContracts } = params

  const seqInboxContract = SequencerInbox__factory.connect(
    config.contracts.sequencerInbox,
    l1Rpc
  )

  // make sure the impl was updated
  if (
    (await getProxyImpl(l1Rpc, config.contracts.sequencerInbox)) !==
    deployedContracts.seqInbox
  ) {
    throw new Error('SequencerInbox was not upgraded')
  }

  // check delay buffer parameters
  const buffer = await seqInboxContract.buffer()

  if (!buffer.bufferBlocks.eq(config.settings.bufferConfig.max)) {
    throw new Error('bufferBlocks does not match')
  }
  if (!buffer.max.eq(config.settings.bufferConfig.max)) {
    throw new Error('max does not match')
  }
  if (!buffer.threshold.eq(config.settings.bufferConfig.threshold)) {
    throw new Error('threshold does not match')
  }
  if (
    !buffer.replenishRateInBasis.eq(
      config.settings.bufferConfig.replenishRateInBasis
    )
  ) {
    throw new Error('replenishRateInBasis does not match')
  }

  // check rollup was set
  if ((await seqInboxContract.rollup()) !== newRollup.address) {
    throw new Error('SequencerInbox rollup address does not match')
  }
}

async function checkInbox(params: VerificationParams) {
  const { l1Rpc, config, deployedContracts } = params

  // make sure the impl was updated
  if (
    (await getProxyImpl(l1Rpc, config.contracts.inbox)) !==
    deployedContracts.inbox
  ) {
    throw new Error('Inbox was not upgraded')
  }
}

async function checkRollupEventInbox(
  params: VerificationParams,
  newRollup: RollupUserLogic
) {
  const { l1Rpc, config, deployedContracts } = params

  const rollupEventInboxContract = RollupEventInbox__factory.connect(
    config.contracts.rollupEventInbox,
    l1Rpc
  )

  // make sure the impl was updated
  if (
    (await getProxyImpl(l1Rpc, config.contracts.rollupEventInbox)) !==
    deployedContracts.rei
  ) {
    throw new Error('RollupEventInbox was not upgraded')
  }

  // make sure rollup was set
  if ((await rollupEventInboxContract.rollup()) !== newRollup.address) {
    throw new Error('RollupEventInbox rollup address does not match')
  }
}

async function checkOutbox(
  params: VerificationParams,
  newRollup: RollupUserLogic
) {
  const { l1Rpc, config, deployedContracts } = params

  const outboxContract = Outbox__factory.connect(config.contracts.outbox, l1Rpc)

  // make sure the impl was updated
  if (
    (await getProxyImpl(l1Rpc, config.contracts.outbox)) !==
    deployedContracts.outbox
  ) {
    throw new Error('Outbox was not upgraded')
  }

  // make sure rollup was set
  if ((await outboxContract.rollup()) !== newRollup.address) {
    throw new Error('Outbox rollup address does not match')
  }
}

async function checkBridge(
  params: VerificationParams,
  newRollup: RollupUserLogic
) {
  const { l1Rpc, config, deployedContracts, preUpgradeState } = params
  const bridgeContract = Bridge__factory.connect(config.contracts.bridge, l1Rpc)

  // make sure the impl was updated
  if (
    (await getProxyImpl(l1Rpc, config.contracts.bridge)) !==
    deployedContracts.bridge
  ) {
    throw new Error('Bridge was not upgraded')
  }

  // make sure rollup was set
  if ((await bridgeContract.rollup()) !== newRollup.address) {
    throw new Error('Bridge rollup address does not match')
  }

  // make sure allowed inbox and outbox list is unchanged
  const { inboxes, outboxes } =
    await getAllowedInboxesOutboxesFromBridge(bridgeContract)
  if (JSON.stringify(inboxes) !== JSON.stringify(preUpgradeState.inboxes)) {
    throw new Error('Allowed inbox list has changed')
  }
  if (JSON.stringify(outboxes) !== JSON.stringify(preUpgradeState.outboxes)) {
    throw new Error('Allowed outbox list has changed')
  }

  // make sure the sequencer inbox is unchanged
  if (
    (await bridgeContract.sequencerInbox()) !== config.contracts.sequencerInbox
  ) {
    throw new Error('Sequencer inbox has changed')
  }
}

async function checkOldRollup(params: VerificationParams) {
  const { l1Rpc, config, deployedContracts, preUpgradeState } = params

  const oldRollupContract = new Contract(
    config.contracts.rollup,
    OldRollupAbi,
    l1Rpc
  )

  // ensure the old rollup is paused
  if (!(await oldRollupContract.paused())) {
    throw new Error('Old rollup is not paused')
  }

  // ensure there are no stakers
  if (!(await oldRollupContract.stakerCount()).eq(0)) {
    throw new Error('Old rollup has stakers')
  }

  // ensure that the old stakers are now zombies
  for (const staker of preUpgradeState.stakers) {
    if (!(await oldRollupContract.isZombie(staker))) {
      throw new Error('Old staker is not a zombie')
    }
  }

  // ensure old rollup was upgraded
  if (
    (await getProxyImpl(l1Rpc, config.contracts.rollup, true)) !==
    getAddress(deployedContracts.oldRollupUser)
  ) {
    throw new Error('Old rollup was not upgraded')
  }
}

async function checkInitialAssertion(
  params: VerificationParams,
  newRollup: RollupUserLogic,
  newEdgeChallengeManager: EdgeChallengeManager
) {
  const { config, l1Rpc } = params

  const bridgeContract = Bridge__factory.connect(config.contracts.bridge, l1Rpc)

  const latestConfirmed = await newRollup.latestConfirmed()

  await newRollup.validateConfig(latestConfirmed, {
    wasmModuleRoot: params.preUpgradeState.wasmModuleRoot,
    requiredStake: config.settings.stakeAmt,
    challengeManager: newEdgeChallengeManager.address,
    confirmPeriodBlocks: config.settings.confirmPeriodBlocks,
    nextInboxPosition: await bridgeContract.sequencerMessageCount(),
  })
}

async function checkNewRollup(
  params: VerificationParams,
  newRollup: RollupUserLogic,
  newEdgeChallengeManager: EdgeChallengeManager
) {
  const { config, deployedContracts, preUpgradeState } = params

  // check bridge
  if (
    getAddress(await newRollup.bridge()) != getAddress(config.contracts.bridge)
  ) {
    throw new Error('Bridge address does not match')
  }

  // check rei
  if (
    getAddress(await newRollup.rollupEventInbox()) !=
    getAddress(config.contracts.rollupEventInbox)
  ) {
    throw new Error('RollupEventInbox address does not match')
  }

  // check inbox
  if (
    getAddress(await newRollup.inbox()) != getAddress(config.contracts.inbox)
  ) {
    throw new Error('Inbox address does not match')
  }

  // check outbox
  if (
    getAddress(await newRollup.outbox()) != getAddress(config.contracts.outbox)
  ) {
    throw new Error('Outbox address does not match')
  }

  // check challengeManager
  if (
    getAddress(await newRollup.challengeManager()) !==
    newEdgeChallengeManager.address
  ) {
    throw new Error('ChallengeManager address does not match')
  }

  // chainId
  if (!(await newRollup.chainId()).eq(config.settings.chainId)) {
    throw new Error('Chain ID does not match')
  }

  // wasmModuleRoot
  if ((await newRollup.wasmModuleRoot()) !== preUpgradeState.wasmModuleRoot) {
    throw new Error('Wasm module root does not match')
  }

  // challengeGracePeriodBlocks
  if (
    !(await newRollup.challengeGracePeriodBlocks()).eq(
      config.settings.challengeGracePeriodBlocks
    )
  ) {
    throw new Error('Challenge grace period blocks does not match')
  }

  // loserStakeEscrow
  if (
    getAddress(await newRollup.loserStakeEscrow()) !==
    getAddress(config.contracts.l1Timelock)
  ) {
    throw new Error('Loser stake escrow address does not match')
  }

  // check initial assertion TODO
  await checkInitialAssertion(params, newRollup, newEdgeChallengeManager)

  // check validator whitelist disabled
  if (
    (await newRollup.validatorWhitelistDisabled()) !==
    config.settings.disableValidatorWhitelist
  ) {
    throw new Error('Validator whitelist disabled does not match')
  }

  // make sure all validators are set
  for (const val of config.validators) {
    if (!(await newRollup.isValidator(val))) {
      throw new Error('Validator not set')
    }
  }

  // check stake token address
  if (
    getAddress(await newRollup.stakeToken()) !=
    getAddress(config.settings.stakeToken)
  ) {
    throw new Error('Stake token address does not match')
  }

  // check confirm period blocks
  if (
    !(await newRollup.confirmPeriodBlocks()).eq(
      config.settings.confirmPeriodBlocks
    )
  ) {
    throw new Error('Confirm period blocks does not match')
  }

  // check base stake
  if (!(await newRollup.baseStake()).eq(config.settings.stakeAmt)) {
    throw new Error('Base stake does not match')
  }

  // check fast confirmer
  if (config.settings.anyTrustFastConfirmer.length != 0) {
    if (
      getAddress(await newRollup.anyTrustFastConfirmer()) !==
      getAddress(config.settings.anyTrustFastConfirmer)
    ) {
      throw new Error('Any trust fast confirmer does not match')
    }
  }
}

async function checkNewChallengeManager(
  params: VerificationParams,
  newRollup: RollupUserLogic,
  edgeChallengeManager: EdgeChallengeManager
) {
  const { config, deployedContracts } = params

  // check assertion chain
  if (
    getAddress(await edgeChallengeManager.assertionChain()) !=
    getAddress(newRollup.address)
  ) {
    throw new Error('Assertion chain address does not match')
  }

  // check challenge period blocks
  if (
    !(await edgeChallengeManager.challengePeriodBlocks()).eq(
      config.settings.challengePeriodBlocks
    )
  ) {
    throw new Error('Challenge period blocks does not match')
  }

  // check osp entry
  if (
    getAddress(await edgeChallengeManager.oneStepProofEntry()) !=
    getAddress(deployedContracts.osp)
  ) {
    throw new Error('OSP address does not match')
  }

  // check level heights
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

  // check stake token address
  if (
    getAddress(await edgeChallengeManager.stakeToken()) !=
    getAddress(config.settings.stakeToken)
  ) {
    throw new Error('Stake token address does not match')
  }

  // check mini stake amounts
  for (let i = 0; i < config.settings.miniStakeAmounts.length; i++) {
    if (
      !(await edgeChallengeManager.stakeAmounts(i)).eq(
        config.settings.miniStakeAmounts[i]
      )
    ) {
      throw new Error('Mini stake amount does not match')
    }
  }

  // check excess stake receiver
  if (
    (await edgeChallengeManager.excessStakeReceiver()) !==
    config.contracts.l1Timelock
  ) {
    throw new Error('Excess stake receiver does not match')
  }

  // check num bigstep levels
  if (
    (await edgeChallengeManager.NUM_BIGSTEP_LEVEL()) !==
    config.settings.numBigStepLevel
  ) {
    throw new Error('Number of big step level does not match')
  }
}

async function getProxyImpl(
  l1Rpc: JsonRpcProvider,
  proxyAddr: string,
  secondary = false
) {
  const primarySlot =
    '0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc'
  const secondarySlot =
    '0x2b1dbce74324248c222f0ec2d5ed7bd323cfc425b336f0253c5ccfda7265546d'
  const val = await l1Rpc.getStorageAt(
    proxyAddr,
    secondary ? secondarySlot : primarySlot
  )
  return getAddress('0x' + val.slice(26))
}

async function getAllowedInboxesOutboxesFromBridge(bridge: Bridge) {
  const inboxes: string[] = []
  const outboxes: string[] = []

  for (let i = 0; ; i++) {
    try {
      inboxes.push(await bridge.allowedDelayedInboxList(i))
    } catch (e: any) {
      if (e.code !== 'CALL_EXCEPTION') {
        throw e
      }
      break
    }
  }

  for (let i = 0; ; i++) {
    try {
      outboxes.push(await bridge.allowedOutboxList(i))
    } catch (e: any) {
      if (e.code !== 'CALL_EXCEPTION') {
        throw e
      }
      break
    }
  }

  return {
    inboxes,
    outboxes,
  }
}

async function main() {
  const l1RpcVal = process.env.L1_RPC_URL
  if (!l1RpcVal) {
    throw new Error('L1_RPC_URL env variable not set')
  }
  const l1Rpc = new ethers.providers.JsonRpcProvider(
    l1RpcVal
  ) as JsonRpcProvider

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
  if (!deployedContracts.boldAction) {
    throw new Error('No boldAction contract deployed')
  }

  const preUpgradeState = await getPreUpgradeState(l1Rpc, config)
  const receipt = await perform(l1Rpc, config, deployedContracts)
  await verifyPostUpgrade({
    l1Rpc,
    config,
    deployedContracts,
    preUpgradeState,
    receipt,
  })
}

main().then(() => console.log('Done.'))
