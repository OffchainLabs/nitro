module.exports = async hre => {
  const { deployments, getNamedAccounts, ethers } = hre
  const { deploy } = deployments
  const { deployer } = await getNamedAccounts()

  const inboxDeployResult = await deploy('InboxStub', {
    from: deployer,
    args: [],
  })

  const bridge = await ethers.getContract('BridgeStub')
  const inbox = await ethers.getContract('InboxStub')

  if (inboxDeployResult.newlyDeployed) {
    await bridge.setDelayedInbox(inbox.address, true)
    await inbox.initialize(bridge.address, ethers.constants.AddressZero)
  }
}

module.exports.tags = ['InboxStub', 'test']
module.exports.dependencies = ['BridgeStub']
