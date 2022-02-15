import "@nomiclabs/hardhat-waffle";
import "hardhat-deploy";
import "@nomiclabs/hardhat-ethers";
import "@typechain/hardhat";
import "solidity-coverage"

/**
 * @type import('hardhat/config').HardhatUserConfig
 */
module.exports = {
    solidity: {
        version: "0.8.6",
        settings: {
          optimizer: {
            enabled: true,
            runs: 100,
          },
        },
    },
    paths: {
        sources: "./src",
    },
    namedAccounts: {
        deployer: {
            default: 0,
        },
    },
    networks: {
        geth: {
            url: "http://localhost:8545"
        }
    },
    mocha: {
        timeout: 0
    }
};
