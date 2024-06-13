import { parseEther } from 'ethers/lib/utils'
import { Config } from '../../common'

export const arb1: Config = {
  contracts: {
    // the l1Timelock does not actually need to be the timelock
    // it is only used to set the excess stake receiver / loser stake escrow
    // TODO: change this to a fee router before real deployment
    l1Timelock: '0xE6841D92B0C345144506576eC13ECf5103aC7f49',
    rollup: '0x5eF0D09d1E6204141B4d37530808eD19f60FBa35',
    bridge: '0x8315177aB297bA92A06054cE80a67Ed4DBd7ed3a',
    sequencerInbox: '0x1c479675ad559DC151F6Ec7ed3FbF8ceE79582B6',
    rollupEventInbox: '0x57Bd336d579A51938619271a7Cc137a46D0501B1',
    outbox: '0x0B9857ae2D4A3DBe74ffE1d7DF045bb7F96E4840',
    inbox: '0x4Dbd4fc535Ac27206064B68FfCf827b0A60BAB3f',
    upgradeExecutor: '0x3ffFbAdAF827559da092217e474760E2b2c3CeDd',
  },
  proxyAdmins: {
    outbox: '0x554723262467f125ac9e1cdfa9ce15cc53822dbd',
    inbox: '0x554723262467f125ac9e1cdfa9ce15cc53822dbd',
    bridge: '0x554723262467f125ac9e1cdfa9ce15cc53822dbd',
    rei: '0x554723262467f125ac9e1cdfa9ce15cc53822dbd',
    seqInbox: '0x554723262467f125ac9e1cdfa9ce15cc53822dbd',
  },
  settings: {
    challengeGracePeriodBlocks: 14400, // 2 days
    confirmPeriodBlocks: 45818, // same as old rollup, ~6.4 days
    challengePeriodBlocks: 45818, // same as confirm period
    stakeToken: '0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2', // WETH
    stakeAmt: parseEther('3600'),
    miniStakeAmounts: [parseEther('0'), parseEther('555'), parseEther('79')],
    chainId: 42161,
    anyTrustFastConfirmer: '0x0000000000000000000000000000000000000000',
    disableValidatorWhitelist: true,
    blockLeafSize: 1048576, // todo below
    bigStepLeafSize: 512,
    smallStepLeafSize: 128,
    numBigStepLevel: 1,
    maxDataSize: 117964,
    isDelayBufferable: true,
    bufferConfig: {
      max: 14400,
      threshold: 300,
      replenishRateInBasis: 500,
    },
  },
  validators: [
    '0x0ff813f6bd577c3d1cdbe435bac0621be6ae34b4',
    '0x54c0d3d6c101580db3be8763a2ae2c6bb9dc840c',
    '0x56d83349c2b8dcf74d7e92d5b6b33d0badd52d78',
    '0x610aa279989f440820e14248bd3879b148717974',
    '0x6fb914de4653ec5592b7c15f4d9466cbd03f2104',
    '0x758c6bb08b3ea5889b5cddbdef9a45b3a983c398',
    '0x7cf3d537733f6ba4183a833c9b021265716ce9d0',
    '0x83215480db2c6a7e56f9e99ef93ab9b36f8a3dd5',
    '0xab1a39332e934300ebcc57b5f95ca90631a347ff',
    '0xb0cb1384e3f4a9a9b2447e39b05e10631e1d34b0',
    '0xddf2f71ab206c0138a8eceeb54386567d5abf01e',
    '0xf59caf75e8a4bfba4e6e07ad86c7e498e4d2519b',
    '0xf8d3e1cf58386c92b27710c6a0d8a54c76bc6ab5',
  ],
}
