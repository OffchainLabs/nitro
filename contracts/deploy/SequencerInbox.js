module.exports = async hre => {
  const { deployments, getNamedAccounts, ethers } = hre
  const { deploy } = deployments
  const { deployer } = await getNamedAccounts()

  await deploy('SequencerInbox', { from: deployer, args: [] })
}

module.exports.tags = ['SequencerInbox']
module.exports.dependencies = []
