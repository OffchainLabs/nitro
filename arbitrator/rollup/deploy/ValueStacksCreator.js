module.exports = async (hre) => {
  const { deployments, getNamedAccounts } = hre;
  const { deploy } = deployments;
  const { deployer } = await getNamedAccounts();

  await deploy("ValueStacks", {
    from: deployer,
    args: [],
  });
};

module.exports.tags = ["ValueStacks", "live", "test"];
module.exports.dependencies = ["Values", "ValueArrays"];
