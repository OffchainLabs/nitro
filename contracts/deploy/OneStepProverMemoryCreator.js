module.exports = async hre => {
  const { deployments, getNamedAccounts } = hre
  const { deploy } = deployments
  const { deployer } = await getNamedAccounts()

  await deploy('OneStepProverMemory', {
    from: deployer,
    args: [],
  })
}

module.exports.tags = ['OneStepProverMemory', 'live', 'test']
module.exports.dependencies = []
