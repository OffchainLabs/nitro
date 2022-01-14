module.exports = async (hre) => {
  const { deployments, getNamedAccounts, ethers } = hre;
  const { deploy } = deployments;
  const { deployer } = await getNamedAccounts();

  await deploy("InboxStub", {from: deployer, args: []});

  const bridge = await ethers.getContract("BridgeStub");
  const inbox = await ethers.getContract("InboxStub");

  const gasOpts = { gasLimit: ethers.utils.hexlify(250000), gasPrice: ethers.utils.parseUnits('5', "gwei") };

  await bridge.setInbox(inbox.address, true, gasOpts);
  await inbox.initialize(bridge.address, gasOpts);
};

module.exports.tags = ["InboxStub", "test"];
module.exports.dependencies = ["BridgeStub"];

