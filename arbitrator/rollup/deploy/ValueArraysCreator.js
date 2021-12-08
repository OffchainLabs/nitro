module.exports = async (hre) => {
  const { deployments, getNamedAccounts } = hre;
  const { deploy } = deployments;
  const { deployer } = await getNamedAccounts();

  await deploy("ValueArrays", {
    from: deployer,
    args: [],
  });
};

module.exports.tags = ["ValueArrays", "live", "test"];
module.exports.dependencies = ["Values"];
