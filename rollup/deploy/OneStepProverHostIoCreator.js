module.exports = async (hre) => {
  const { deployments, getNamedAccounts, ethers } = hre;
  const { deploy } = deployments;
  const { deployer } = await getNamedAccounts();

  await deploy("Bridge", {from: deployer, args: []})
  await deploy("Inbox", {from: deployer, args: []})

  bridge = await ethers.getContract("Bridge", deployer)
  inbox = await ethers.getContract("Inbox", deployer)

  await deploy("SequencerInbox", {from: deployer, args: [bridge.address, deployer]})

  await bridge.setInbox(inbox.address, true);
  await inbox.initialize(bridge.address);

  seqInbox = await ethers.getContract("SequencerInbox", deployer)

  await deploy("OneStepProverHostIo", {
    from: deployer,
    args: [seqInbox.address, bridge.address],
  });
};

module.exports.tags = ["OneStepProverHostIo", "live", "test"];
module.exports.dependencies = ["Machines"];

