require("@nomiclabs/hardhat-waffle");
require("hardhat-deploy");
require("@nomiclabs/hardhat-ethers");

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
};
