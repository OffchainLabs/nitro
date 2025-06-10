/**
 * @type import('hardhat/config').HardhatUserConfig
 */
module.exports = {
  compilers: [
    {
      version: '0.8.24',
      settings: {
        optimizer: {
          enabled: true,
          runs: 100,
        },
        evmVersion: 'cancun',
      },
    },
  ],
  paths: {
    sources: './src',
    artifacts: 'build/contracts',
  },
  typechain: {
    outDir: 'build/types',
    target: 'ethers-v5',
  },
}
