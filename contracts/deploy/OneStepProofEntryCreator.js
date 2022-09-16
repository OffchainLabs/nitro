module.exports = async hre => {
  const { deployments, getNamedAccounts } = hre
  const { deploy } = deployments
  const { deployer } = await getNamedAccounts()

  await deploy('OneStepProofEntry', {
    from: deployer,
    args: [
      (await deployments.get('OneStepProver0')).address,
      (await deployments.get('OneStepProverMemory')).address,
      (await deployments.get('OneStepProverMath')).address,
      (await deployments.get('OneStepProverHostIo')).address,
    ],
  })
}

module.exports.tags = ['OneStepProofEntry']
module.exports.dependencies = [
  'OneStepProver0',
  'OneStepProverMemory',
  'OneStepProverMath',
  'OneStepProverHostIo',
]
