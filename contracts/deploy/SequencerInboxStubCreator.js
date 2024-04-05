import { Toolkit4844 } from '../test/contract/toolkit4844'

module.exports = async hre => {
  const { deployments, getSigners, getNamedAccounts, ethers } = hre
  const { deploy } = deployments
  const { deployer } = await getNamedAccounts()

  const bridge = await ethers.getContract('BridgeStub')
  const reader4844 = await Toolkit4844.deployReader4844(
    await ethers.getSigner(deployer)
  )
  const maxTime = {
    delayBlocks: 10000,
    futureBlocks: 10000,
    delaySeconds: 10000,
    futureSeconds: 10000,
  }
  await deploy('SequencerInboxStub', {
    from: deployer,
    args: [
      bridge.address,
      deployer,
      maxTime,
      117964,
      reader4844.address,
      false,
    ],
  })
}

module.exports.tags = ['SequencerInboxStub', 'test']
module.exports.dependencies = ['BridgeStub']
