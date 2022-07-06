import '@nomiclabs/hardhat-waffle'
import 'hardhat-deploy'
import '@nomiclabs/hardhat-ethers'
import '@typechain/hardhat'
import 'solidity-coverage'
import 'hardhat-gas-reporter'
import prodConfig from "./hardhat.prod-config"

const solidity = {
  compilers: [
    {
      version: "0.8.9",
      settings: {
        optimizer: {
          enabled: true,
          runs: 100,
        },
      },
    },
  ],
  overrides: {},
};

if (process.env["INTERFACE_TESTER_SOLC_VERSION"]) {
  solidity.compilers.push({
    version: process.env["INTERFACE_TESTER_SOLC_VERSION"],
    settings: {
      optimizer: {
        enabled: true,
        runs: 100,
      },
    },
  });
  solidity.overrides = {
    "src/test-helpers/InterfaceCompatibilityTester.sol": {
      version: process.env["INTERFACE_TESTER_SOLC_VERSION"],
      settings: {
        optimizer: {
          enabled: true,
          runs: 100,
        },
      },
    },
  };
}

/**
 * @type import('hardhat/config').HardhatUserConfig
 */
module.exports = {
  ...prodConfig,
  solidity,
  namedAccounts: {
    deployer: {
      default: 0,
    },
  },
  networks: {
    hardhat: {
      chainId: 1338,
      throwOnTransactionFailures: true,
      allowUnlimitedContractSize: true,
      accounts: {
        accountsBalance: "1000000000000000000000000000",
      },
      blockGasLimit: 200000000,
      // mining: {
      //   auto: false,
      //   interval: 1000,
      // },
      forking: {
        url: "https://mainnet.infura.io/v3/" + process.env["INFURA_KEY"],
        enabled: process.env["SHOULD_FORK"] === "1",
      },
    },
    geth: {
      url: "http://localhost:8545",
    },
  },
  mocha: {
    timeout: 0,
  },
  gasReporter: {
    enabled: process.env.DISABLE_GAS_REPORTER ? false : true,
  },
  typechain: {
    outDir: "build/types",
    target: "ethers-v5",
  },
};
