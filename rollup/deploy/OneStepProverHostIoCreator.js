module.exports = async (hre) => {
  const { deployments, getNamedAccounts } = hre;
  const { deploy } = deployments;
  const { deployer } = await getNamedAccounts();

  await deploy("OneStepProverHostIo", {
    from: deployer,
    args: ["0x0000000000000000000000000000000000000000", "0x0000000000000000000000000000000000000000"],
  });
};

module.exports.tags = ["OneStepProverHostIo", "live", "test"];
module.exports.dependencies = ["Machines"];

