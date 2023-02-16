module.exports = async hre => {
  const { deployments, getNamedAccounts } = hre
  const { deploy } = deployments
  const { deployer } = await getNamedAccounts()

  await deploy('OneStepProverMath', {
    from: deployer,
    args: [],
  })
}

module.exports.tags = ['OneStepProverMath', 'live', 'test']
module.exports.dependencies = []
