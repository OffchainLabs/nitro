module.exports = async hre => {
  const { deployments, getNamedAccounts, ethers } = hre
  const { deploy } = deployments
  const { deployer } = await getNamedAccounts()

  const blobBasefeeReader = await ethers.getContract('BlobBasefeeReader')
  const dataHashReader = await ethers.getContract('DataHashReader')

  await deploy('SequencerInbox', { from: deployer, args: [117964] })
}

module.exports.tags = ['SequencerInbox']
module.exports.dependencies = []
