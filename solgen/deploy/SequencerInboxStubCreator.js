module.exports = async (hre) => {
  const { deployments, getNamedAccounts, ethers } = hre;
  const { deploy } = deployments;
  const { deployer } = await getNamedAccounts();

  const bridge = await ethers.getContract("BridgeStub");

  await deploy("SequencerInboxStub", {from: deployer, args: [bridge.address, deployer]})
};

module.exports.tags = ["SequencerInboxStub", "test"];
module.exports.dependencies = ["BridgeStub"];

