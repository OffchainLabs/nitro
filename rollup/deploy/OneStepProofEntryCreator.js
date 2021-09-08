module.exports = async (hre) => {
  const { deployments, getNamedAccounts } = hre;
  const { deploy } = deployments;
  const { deployer } = await getNamedAccounts();

  await deploy("OneStepProofEntry", {
    from: deployer,
    args: [(await deployments.get("OneStepProver0")).address],
  });
};

module.exports.tags = ["OneStepProofEntry", "live", "test"];
module.exports.dependencies = ["Machines", "OneStepProver0"];

