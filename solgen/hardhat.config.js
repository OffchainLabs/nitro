require("@nomiclabs/hardhat-waffle");
require("hardhat-deploy");
require("@nomiclabs/hardhat-ethers");

/**
 * @type import('hardhat/config').HardhatUserConfig
 */
module.exports = {
    solidity: {
        compilers: [
            { version: "0.8.6", },
            { version: "0.7.5", },
        ],
    },
    paths: {
        sources: "./src",
    },
    namedAccounts: {
        deployer: {
            default: 0,
        },
    },
};
