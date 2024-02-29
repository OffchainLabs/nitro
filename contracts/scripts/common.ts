import { BigNumber, providers } from 'ethers'
import { parseEther } from 'ethers/lib/utils'
import fs from 'fs'
export interface DeployedContracts {
  bridge: string
  seqInbox: string
  rei: string
  outbox: string
  oldRollupUser: string
  newRollupUser: string
  newRollupAdmin: string
  challengeManager: string
  boldAction: string
  rollupReader: string
  preImageHashLookup: string
  prover0: string
  proverMem: string
  proverMath: string
  proverHostIo: string
  osp: string
  newEdgeChallengeManager?: string
}

export const getJsonFile = (fileLocation: string) => {
  return JSON.parse(fs.readFileSync(fileLocation).toString())
}

export const getConfig = async (
  configLocation: string,
  l1Rpc: providers.Provider
): Promise<Config> => {
  const config = getJsonFile(configLocation) as RawConfig
  return await validateConfig(config, l1Rpc)
}

export interface Config {
  contracts: {
    l1Timelock: string
    rollup: string
    bridge: string
    sequencerInbox: string
    rollupEventInbox: string
    outbox: string
    inbox: string
    upgradeExecutor: string
  }
  proxyAdmins: {
    outbox: string
    bridge: string
    rei: string
    seqInbox: string
  }
  settings: {
    challengeGracePeriodBlocks: number,
    confirmPeriodBlocks: number
    challengePeriodBlocks: number
    stakeToken: string
    stakeAmt: BigNumber
    miniStakeAmounts: BigNumber[]
    chainId: number
    anyTrustFastConfirmer: string
    disableValidatorWhitelist: boolean
    maxDataSize: number
    blockLeafSize: number
    bigStepLeafSize: number
    smallStepLeafSize: number
    numBigStepLevel: number
  }
  validators: string[]
}

export type RawConfig = Omit<Config, 'settings'> & {
  settings: Omit<Config['settings'], 'stakeAmt' | 'miniStakeAmounts'> & {
    stakeAmt: string
    miniStakeAmounts: string[]
  }
}

export const validateConfig = async (
  config: RawConfig,
  l1Rpc: providers.Provider
): Promise<Config> => {
  // check all the config.contracts exist
  if ((await l1Rpc.getCode(config.contracts.l1Timelock)).length <= 2) {
    throw new Error('l1Timelock address is not a contract')
  }
  if ((await l1Rpc.getCode(config.contracts.rollup)).length <= 2) {
    throw new Error('rollup address is not a contract')
  }
  if ((await l1Rpc.getCode(config.contracts.bridge)).length <= 2) {
    throw new Error('bridge address is not a contract')
  }
  if ((await l1Rpc.getCode(config.contracts.sequencerInbox)).length <= 2) {
    throw new Error('sequencerInbox address is not a contract')
  }
  if ((await l1Rpc.getCode(config.contracts.rollupEventInbox)).length <= 2) {
    throw new Error('rollupEventInbox address is not a contract')
  }
  if ((await l1Rpc.getCode(config.contracts.outbox)).length <= 2) {
    throw new Error('outbox address is not a contract')
  }
  if ((await l1Rpc.getCode(config.contracts.inbox)).length <= 2) {
    throw new Error('inbox address is not a contract')
  }
  if ((await l1Rpc.getCode(config.contracts.upgradeExecutor)).length <= 2) {
    throw new Error('upgradeExecutor address is not a contract')
  }

  // check all the config.proxyAdmins exist
  if ((await l1Rpc.getCode(config.proxyAdmins.outbox)).length <= 2) {
    throw new Error('outbox proxy admin address is not a contract')
  }
  if ((await l1Rpc.getCode(config.proxyAdmins.bridge)).length <= 2) {
    throw new Error('bridge proxy admin address is not a contract')
  }
  if ((await l1Rpc.getCode(config.proxyAdmins.rei)).length <= 2) {
    throw new Error('rei proxy admin address is not a contract')
  }
  if ((await l1Rpc.getCode(config.proxyAdmins.seqInbox)).length <= 2) {
    throw new Error('seqInbox proxy admin address is not a contract')
  }

  // check all the settings exist
  if (config.settings.confirmPeriodBlocks == 0) {
    throw new Error('confirmPeriodBlocks is 0')
  }
  if (config.settings.stakeToken.length == 0) {
    throw new Error('stakeToken address is empty')
  }
  if (config.settings.chainId == 0) {
    throw new Error('chainId is 0')
  }
  if (config.settings.blockLeafSize == 0) {
    throw new Error('blockLeafSize is 0')
  }
  if (config.settings.bigStepLeafSize == 0) {
    throw new Error('bigStepLeafSize is 0')
  }
  if (config.settings.smallStepLeafSize == 0) {
    throw new Error('smallStepLeafSize is 0')
  }
  if (config.settings.numBigStepLevel == 0) {
    throw new Error('numBigStepLevel is 0')
  }

  const stakeAmount = BigNumber.from(config.settings.stakeAmt)
  // check it's more than 1 eth
  if (stakeAmount.lt(parseEther('1'))) {
    throw new Error('stakeAmt is less than 1 eth')
  }
  const miniStakeAmounts = config.settings.miniStakeAmounts.map(BigNumber.from)

  if (miniStakeAmounts.length !== config.settings.numBigStepLevel + 2) {
    throw new Error('miniStakeAmts length is not numBigStepLevel + 2')
  }
  if (miniStakeAmounts.some((amt) => amt.lt(parseEther('0.1')))) {
    throw new Error('miniStakeAmt is less than 0.1 eth')
  }

  if (config.validators.length == 0) {
    throw new Error('no validators')
  }

  return {
    ...config,
    settings: {
      ...config.settings,
      stakeAmt: stakeAmount,
      miniStakeAmounts,
    },
  }
}
