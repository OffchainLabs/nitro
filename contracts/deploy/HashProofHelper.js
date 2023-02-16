module.exports = async hre => {
  const { deployments, getNamedAccounts } = hre
  const { deploy } = deployments
  const { deployer } = await getNamedAccounts()

  await deploy('HashProofHelper', {
    from: deployer,
    args: [],
  })
}

module.exports.tags = ['HashProofHelper', 'test', 'live']
module.exports.dependencies = []
