import '@nomiclabs/hardhat-waffle'
import 'hardhat-deploy'
import '@nomiclabs/hardhat-ethers'
import '@nomiclabs/hardhat-etherscan'
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
    mainnet: {
      url: "https://mainnet.infura.io/v3/" + process.env["INFURA_KEY"],
      accounts: process.env["MAINNET_PRIVKEY"] ? [process.env["MAINNET_PRIVKEY"]] : [],
    },
    goerli: {
      url: "https://goerli.infura.io/v3/" + process.env["INFURA_KEY"],
      accounts: process.env["DEVNET_PRIVKEY"] ? [process.env["DEVNET_PRIVKEY"]] : [],
    },
    rinkeby: {
      url: "https://rinkeby.infura.io/v3/" + process.env["INFURA_KEY"],
      accounts: process.env["DEVNET_PRIVKEY"] ? [process.env["DEVNET_PRIVKEY"]] : [],
    },
    arbRinkeby: {
      url: "https://rinkeby.arbitrum.io/rpc",
      accounts: process.env["DEVNET_PRIVKEY"] ? [process.env["DEVNET_PRIVKEY"]] : [],
    },
    arbGoerliRollup: {
      url: "https://goerli-rollup.arbitrum.io/rpc",
      accounts: process.env["DEVNET_PRIVKEY"] ? [process.env["DEVNET_PRIVKEY"]] : [],
    },
    arb1: {
      url: "https://arb1.arbitrum.io/rpc",
      accounts: process.env["MAINNET_PRIVKEY"] ? [process.env["MAINNET_PRIVKEY"]] : [],
    },
    nova: {
      url: "https://nova.arbitrum.io/rpc",
      accounts: process.env["MAINNET_PRIVKEY"] ? [process.env["MAINNET_PRIVKEY"]] : [],
    },
    geth: {
      url: "http://localhost:8545",
    },
  },
  etherscan: {
    apiKey: {
      mainnet: process.env["ETHERSCAN_API_KEY"],
      goerli: process.env["ETHERSCAN_API_KEY"],
      rinkeby: process.env["ETHERSCAN_API_KEY"],
      arbitrumOne: process.env["ARBISCAN_API_KEY"],
      arbitrumTestnet: process.env["ARBISCAN_API_KEY"],
      nova: "0",
      arbGoerliRollup: "0",
    },
    customChains: [
      {
        network: "nova",
        chainId: 42170,
        urls: {
          apiURL: "https://nova-explorer.arbitrum.io/api",
          browserURL: "https://nova-explorer.arbitrum.io/",
        },
      },
      {
        network: "arbGoerliRollup",
        chainId: 421613,
        urls: {
          apiURL: "https://goerli-rollup-explorer.arbitrum.io/api",
          browserURL: "https://goerli-rollup-explorer.arbitrum.io/",
        },
      },
    ],
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
