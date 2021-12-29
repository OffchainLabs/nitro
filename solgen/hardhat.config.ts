import "@nomiclabs/hardhat-waffle";
import "hardhat-deploy";
import "@nomiclabs/hardhat-ethers";
import "@typechain/hardhat";
import "solidity-coverage"

/**
 * @type import('hardhat/config').HardhatUserConfig
 */
module.exports = {
    solidity: "0.8.6",
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
