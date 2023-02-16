module.exports = async hre => {
  const { deployments, getNamedAccounts } = hre
  const { deploy } = deployments
  const { deployer } = await getNamedAccounts()

  await deploy('OneStepProver0', {
    from: deployer,
    args: [],
  })
}

module.exports.tags = ['OneStepProver0', 'live', 'test']
module.exports.dependencies = []
