import { setupNetworks, config } from './testSetup'
import * as fs from 'fs'

async function main() {
  const { l1Network, l2Network } = await setupNetworks(
    config.ethUrl,
    config.arbUrl
  )

  fs.writeFileSync(
    './files/local/network.json',
    JSON.stringify({ l1Network, l2Network }, null, 2)
  )
  console.log('network.json updated')
}

main().then(() => console.log('Done.'))
