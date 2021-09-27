module.exports = async (hre) => {
  const { deployments, getNamedAccounts } = hre;
  const { deploy } = deployments;
  const { deployer } = await getNamedAccounts();

  await deploy("OneStepProverHostIo", {
    from: deployer,
    args: [],
  });
};

module.exports.tags = ["OneStepProverHostIo", "live", "test"];
module.exports.dependencies = ["Machines"];

