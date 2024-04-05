import '@nomiclabs/hardhat-ethers'
import { createRollup } from './rollupCreation'

async function main() {
  await createRollup()
}

main()
  .then(() => process.exit(0))
  .catch((error: Error) => {
    console.error(error)
    process.exit(1)
  })
