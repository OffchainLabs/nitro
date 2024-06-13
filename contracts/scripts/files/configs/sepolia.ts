import { parseEther } from 'ethers/lib/utils'
import { Config } from '../../common'

export const sepolia: Config = {
  contracts: {
    l1Timelock: '0x6EC62D826aDc24AeA360be9cF2647c42b9Cdb19b',
    rollup: '0xd80810638dbDF9081b72C1B33c65375e807281C8',
    bridge: '0x38f918D0E9F1b721EDaA41302E399fa1B79333a9',
    sequencerInbox: '0x6c97864CE4bEf387dE0b3310A44230f7E3F1be0D',
    rollupEventInbox: '0xD5B196dd7EC4D823ff5F695536c61f7c8E642B94',
    outbox: '0x65f07C7D521164a4d5DaC6eB8Fac8DA067A3B78F',
    inbox: '0xaAe29B0366299461418F5324a79Afc425BE5ae21',
    upgradeExecutor: '0x5FEe78FE9AD96c1d8557C6D6BB22Eb5A61eeD315',
  },
  proxyAdmins: {
    outbox: '0xdd63bcaa89d7c3199ef220c1dd59c49f821078b8',
    inbox: '0xdd63bcaa89d7c3199ef220c1dd59c49f821078b8',
    bridge: '0xdd63bcaa89d7c3199ef220c1dd59c49f821078b8',
    rei: '0xdd63bcaa89d7c3199ef220c1dd59c49f821078b8',
    seqInbox: '0xdd63bcaa89d7c3199ef220c1dd59c49f821078b8',
  },
  settings: {
    challengeGracePeriodBlocks: 14400,
    confirmPeriodBlocks: 50400,
    challengePeriodBlocks: 51600,
    stakeToken: '0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2',
    stakeAmt: parseEther('1'),
    miniStakeAmounts: [
      parseEther('6'),
      parseEther('5'),
      parseEther('4'),
      parseEther('3'),
      parseEther('2'),
      parseEther('1'),
    ],
    chainId: 42161,
    anyTrustFastConfirmer: '0x0000000000000000000000000000000000000000',
    disableValidatorWhitelist: false,
    blockLeafSize: 1048576,
    bigStepLeafSize: 512,
    smallStepLeafSize: 128,
    numBigStepLevel: 4,
    maxDataSize: 117964,
    isDelayBufferable: true,
    bufferConfig: {
      max: 14400,
      threshold: 300,
      replenishRateInBasis: 500,
    },
  },
  validators: [
    '0x8a8f0a24d7e58a76FC8F77bb68C7c902b91e182e',
    '0x87630025E63A30eCf9Ca9d580d9D95922Fea6aF0',
    '0xC32B93e581db6EBc50C08ce381143A259B92f1ED',
  ],
}
