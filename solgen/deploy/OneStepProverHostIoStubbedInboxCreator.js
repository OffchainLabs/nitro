module.exports = async (hre) => {
  const { deployments, getNamedAccounts, ethers } = hre;
  const { deploy } = deployments;
  const { deployer } = await getNamedAccounts();

  const gasOpts = { gasLimit: ethers.utils.hexlify(250000), gasPrice: ethers.utils.parseUnits('5', "gwei") };

  await deploy("BridgeStub", {from: deployer, args: []})
  await deploy("InboxStub", {from: deployer, args: []})

  const bridge = await ethers.getContract("BridgeStub", deployer)
  const inbox = await ethers.getContract("InboxStub", deployer)

  await deploy("SequencerInboxStub", {from: deployer, args: [bridge.address, deployer]})

  await bridge.setInbox(inbox.address, true, gasOpts);
  await inbox.initialize(bridge.address, gasOpts);

  const seqInbox = await ethers.getContract("SequencerInboxStub", deployer)

  await deploy("OneStepProverHostIo", {
    from: deployer,
    args: [seqInbox.address, bridge.address],
  });
};

module.exports.tags = ["OneStepProverHostIoStubbedInbox", "test"];
module.exports.dependencies = [];

