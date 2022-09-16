module.exports = async hre => {
  const { deployments, getNamedAccounts, ethers } = hre
  const { deploy } = deployments
  const { deployer } = await getNamedAccounts()

  const bridge = await ethers.getContract('BridgeStub')
  const maxTime = {
    delayBlocks: 10000,
    futureBlocks: 10000,
    delaySeconds: 10000,
    futureSeconds: 10000,
  }
  await deploy('SequencerInboxStub', {
    from: deployer,
    args: [bridge.address, deployer, maxTime],
  })
}

module.exports.tags = ['SequencerInboxStub', 'test']
module.exports.dependencies = ['BridgeStub']
