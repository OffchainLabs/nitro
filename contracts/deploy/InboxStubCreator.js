module.exports = async (hre) => {
  const { deployments, getNamedAccounts, ethers } = hre;
  const { deploy } = deployments;
  const { deployer } = await getNamedAccounts();

  await deploy("InboxStub", {from: deployer, args: []});

  const bridge = await ethers.getContract("BridgeStub");
  const inbox = await ethers.getContract("InboxStub");

  await bridge.setInbox(inbox.address, true);
  await inbox.initialize(bridge.address);
};

module.exports.tags = ["InboxStub", "test"];
module.exports.dependencies = ["BridgeStub"];

