import { BigNumber, Contract, ContractFactory, ethers, Signer } from 'ethers'
import {
  BOLDUpgradeAction__factory,
  Bridge__factory,
  EdgeChallengeManager__factory,
  OneStepProofEntry__factory,
  OneStepProver0__factory,
  OneStepProverHostIo__factory,
  OneStepProverMath__factory,
  OneStepProverMemory__factory,
  Outbox__factory,
  RollupAdminLogic__factory,
  RollupEventInbox__factory,
  RollupReader__factory,
  RollupUserLogic__factory,
  SequencerInbox__factory,
  Inbox__factory,
  StateHashPreImageLookup__factory,
  IReader4844__factory,
} from '../build/types'
import { bytecode as Reader4844Bytecode } from '../out/yul/Reader4844.yul/Reader4844.json'
import { DeployedContracts, Config } from './common'
import { AssertionStateStruct } from '../build/types/src/challengeV2/IAssertionChain'
// taken from https://github.com/OffchainLabs/nitro-contracts/blob/210e5b3bc96a513d276deaba90399130a60131d5/src/rollup/RollupUserLogic.sol
import {
  abi as OldRollupAbi,
  bytecode as OldRollupBytecode,
} from './files/OldRollupUserLogic.json'

export const deployDependencies = async (
  signer: Signer,
  maxDataSize: number,
  isUsingFeeToken: boolean,
  isDelayBufferable: boolean,
  log: boolean = false,
): Promise<
  Omit<DeployedContracts, 'boldAction' | 'preImageHashLookup' | 'rollupReader'>
> => {
  const bridgeFac = new Bridge__factory(signer)
  const bridge = await bridgeFac.deploy()
  if (log) {
    console.log(`Bridge implementation deployed at: ${bridge.address}`)
  }

  const contractFactory = new ContractFactory(
    IReader4844__factory.abi,
    Reader4844Bytecode,
    signer
  )
  const reader4844 = await contractFactory.deploy()
  await reader4844.deployed()
  console.log(`Reader4844 deployed at ${reader4844.address}`)

  const seqInboxFac = new SequencerInbox__factory(signer)
  const seqInbox = await seqInboxFac.deploy(maxDataSize, reader4844.address, isUsingFeeToken, isDelayBufferable)
  if (log) {
    console.log(
      `Sequencer inbox implementation deployed at: ${seqInbox.address}`
    )
  }

  const reiFac = new RollupEventInbox__factory(signer)
  const rei = await reiFac.deploy()
  if (log) {
    console.log(`Rollup event inbox implementation deployed at: ${rei.address}`)
  }

  const outboxFac = new Outbox__factory(signer)
  const outbox = await outboxFac.deploy()
  if (log) {
    console.log(`Outbox implementation deployed at: ${outbox.address}`)
  }

  const inboxFac = new Inbox__factory(signer)
  const inbox = await inboxFac.deploy(maxDataSize)
  if (log) {
    console.log(`Inbox implementation deployed at: ${inbox.address}`)
  }

  const oldRollupUserFac = new ContractFactory(
    OldRollupAbi,
    OldRollupBytecode,
    signer
  )
  const oldRollupUser = await oldRollupUserFac.deploy()
  if (log) {
    console.log(`Old rollup user logic deployed at: ${oldRollupUser.address}`)
  }

  const newRollupUserFac = new RollupUserLogic__factory(signer)
  const newRollupUser = await newRollupUserFac.deploy()
  if (log) {
    console.log(`New rollup user logic deployed at: ${newRollupUser.address}`)
  }

  const newRollupAdminFac = new RollupAdminLogic__factory(signer)
  const newRollupAdmin = await newRollupAdminFac.deploy()
  if (log) {
    console.log(`New rollup admin logic deployed at: ${newRollupAdmin.address}`)
  }

  const challengeManagerFac = new EdgeChallengeManager__factory(signer)
  const challengeManager = await challengeManagerFac.deploy()
  if (log) {
    console.log(`Challenge manager deployed at: ${challengeManager.address}`)
  }

  const prover0Fac = new OneStepProver0__factory(signer)
  const prover0 = await prover0Fac.deploy()
  await prover0.deployed()
  if (log) {
    console.log(`Prover0 deployed at: ${prover0.address}`)
  }

  const proverMemFac = new OneStepProverMemory__factory(signer)
  const proverMem = await proverMemFac.deploy()
  await proverMem.deployed()
  if (log) {
    console.log(`Prover mem deployed at: ${proverMem.address}`)
  }

  const proverMathFac = new OneStepProverMath__factory(signer)
  const proverMath = await proverMathFac.deploy()
  await proverMath.deployed()
  if (log) {
    console.log(`Prover math deployed at: ${proverMath.address}`)
  }

  const proverHostIoFac = new OneStepProverHostIo__factory(signer)
  const proverHostIo = await proverHostIoFac.deploy()
  await proverHostIo.deployed()
  if (log) {
    console.log(`Prover host io deployed at: ${proverHostIo.address}`)
  }

  const proofEntryFac = new OneStepProofEntry__factory(signer)
  const proofEntry = await proofEntryFac.deploy(
    prover0.address,
    proverMem.address,
    proverMath.address,
    proverHostIo.address
  )
  await proofEntry.deployed()
  if (log) {
    console.log(`Proof entry deployed at: ${proofEntry.address}`)
  }

  return {
    bridge: bridge.address,
    seqInbox: seqInbox.address,
    rei: rei.address,
    outbox: outbox.address,
    inbox: inbox.address,
    oldRollupUser: oldRollupUser.address,
    newRollupUser: newRollupUser.address,
    newRollupAdmin: newRollupAdmin.address,
    challengeManager: challengeManager.address,
    prover0: prover0.address,
    proverMem: proverMem.address,
    proverMath: proverMath.address,
    proverHostIo: proverHostIo.address,
    osp: proofEntry.address,
  }
}

export const deployBoldUpgrade = async (
  wallet: Signer,
  config: Config,
  log: boolean = false
): Promise<DeployedContracts> => {
  const sequencerInbox = SequencerInbox__factory.connect(config.contracts.sequencerInbox, wallet)
  const isUsingFeeToken = await sequencerInbox.isUsingFeeToken()
  const deployed = await deployDependencies(
    wallet, 
    config.settings.maxDataSize, 
    isUsingFeeToken,
    config.settings.isDelayBufferable,
    log
  )
  const fac = new BOLDUpgradeAction__factory(wallet)
  const boldUpgradeAction = await fac.deploy(
    { ...config.contracts, osp: deployed.osp },
    config.proxyAdmins,
    deployed,
    config.settings
  )
  if (log) {
    console.log(`BOLD upgrade action deployed at: ${boldUpgradeAction.address}`)
  }
  const deployedAndBold = {
    ...deployed,
    boldAction: boldUpgradeAction.address,
    rollupReader: await boldUpgradeAction.ROLLUP_READER(),
    preImageHashLookup: await boldUpgradeAction.PREIMAGE_LOOKUP(),
  }

  return deployedAndBold
}

export const populateLookup = async (
  wallet: Signer,
  rollupAddr: string,
  preImageHashLookupAddr: string,
  rollupReaderAddr: string
) => {
  const oldRollup = new Contract(rollupAddr, OldRollupAbi, wallet.provider)
  const latestConfirmed: number = await oldRollup.latestConfirmed()
  const latestConfirmedLog = await wallet.provider!.getLogs({
    address: rollupAddr,
    fromBlock: 0,
    toBlock: 'latest',
    topics: [
      oldRollup.interface.getEventTopic('NodeCreated'),
      ethers.utils.hexZeroPad(ethers.utils.hexlify(latestConfirmed), 32),
    ],
  })

  if (latestConfirmedLog.length != 1) {
    throw new Error('Could not find latest confirmed node')
  }
  const latestConfirmedEvent = oldRollup.interface.parseLog(
    latestConfirmedLog[0]
  ).args
  const afterState: AssertionStateStruct =
    latestConfirmedEvent.assertion.afterState
  const inboxCount: BigNumber = latestConfirmedEvent.inboxMaxCount

  const lookup = StateHashPreImageLookup__factory.connect(
    preImageHashLookupAddr,
    wallet
  )
  const oldRollupReader = RollupReader__factory.connect(
    rollupReaderAddr,
    wallet
  )
  const node = await oldRollupReader.getNode(latestConfirmed)
  const stateHash = await lookup.stateHash(afterState, inboxCount)
  if (node.stateHash != stateHash) {
    throw new Error(`State hash mismatch ${node.stateHash} != ${stateHash}}`)
  }

  await lookup.set(stateHash, afterState, inboxCount)
}
