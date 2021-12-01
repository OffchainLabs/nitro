module.exports = async (hre) => {
  const { deployments, getNamedAccounts } = hre;
  const { deploy } = deployments;
  const { deployer } = await getNamedAccounts();

  await deploy("Values", {
    from: deployer,
    args: [],
  });
};

module.exports.tags = ["Values", "live", "test"];
