module.exports = async (hre) => {
  const { deployments, getNamedAccounts, ethers } = hre;
  const { deploy } = deployments;
  const { deployer } = await getNamedAccounts();

  const gasOpts = { gasLimit: ethers.utils.hexlify(250000), gasPrice: ethers.utils.parseUnits('5', "gwei") };

  await deploy("Bridge", {from: deployer, args: []})
  await deploy("Inbox", {from: deployer, args: []})

  const bridge = await ethers.getContract("Bridge", deployer)
  const inbox = await ethers.getContract("Inbox", deployer)

  await deploy("SequencerInbox", {from: deployer, args: [bridge.address, deployer]})

  await bridge.setInbox(inbox.address, true, gasOpts);
  await inbox.initialize(bridge.address, gasOpts);

  const seqInbox = await ethers.getContract("SequencerInbox", deployer)

  await deploy("OneStepProverHostIo", {
    from: deployer,
    args: [seqInbox.address, bridge.address],
  });
};

module.exports.tags = ["OneStepProverHostIo", "live", "test"];
module.exports.dependencies = ["Machines"];

