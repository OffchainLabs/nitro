module.exports = async (hre) => {
  const { deployments, getNamedAccounts } = hre;
  const { deploy } = deployments;
  const { deployer } = await getNamedAccounts();

  await deploy("Machines", {
    from: deployer,
    args: [],
  });
};

module.exports.tags = ["Machines", "live", "test"];
module.exports.dependencies = ["ValueStacks"];
