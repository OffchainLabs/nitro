# Start Developing on Arbitrum

Developing on Arbitrum is as easy as developing on Ethereum — literally!

# Querying Information from the Arbitrum Blockchain

To query for information such as the latest block number or NFTs associated with a specific address on the Arbitrum blockchain, simply install the [Alchemy SDK](https://docs.alchemy.com/reference/alchemy-sdk-quickstart), set the RPC endpoint of your target Arbitrum chain, and start requesting information.

See a simple code example:
```
// Javascript
// Setup: npm install alchemy-sdk
const { Alchemy, Network } = require("alchemy-sdk");

const settings = {
  apiKey: "demo", // Can replace with your API Key from https://www.alchemy.com
  network: Network.ARB_MAINNET, // Can replace with ARB_GOERLI
};

const alchemy = new Alchemy(settings);

async function main() {
  const latestBlock = await alchemy.core.getBlockNumber();
  console.log("The latest block number is", latestBlock);
}

main();
```

You can see a list of examples on how to make various queries from the Arbitrum blockchain by going to [Arbitrum SDK Examples](https://docs.alchemy.com/reference/arbitrum-sdk-examples).

Examples include: 
- [Getting Logs from an Arbitrum Transaction](https://docs.alchemy.com/reference/arbitrum-sdk-examples#how-to-get-logs-for-an-arbitrum-transaction)
- [Fetching Historical Transactions on Arbitrum](https://docs.alchemy.com/reference/arbitrum-sdk-examples#how-to-get-logs-for-an-arbitrum-transaction)
- [Subscribing to New Blocks via WebSockets on Arbitrum](https://docs.alchemy.com/reference/arbitrum-sdk-examples#how-to-subscribe-to-new-blocks-on-arbitrum)
- [Making an eth_call on Arbitrum](https://docs.alchemy.com/reference/arbitrum-sdk-examples#how-to-make-an-eth_call-on-arbitrum)

# Deploying Contracts onto Arbitrum 

To deploy contracts onto an Arbitrum chain, simple set the RPC endpoint (see [Public Chains)](public-chains) of your target Arbitrum chain and deploy using your favorite Ethereum development framework;

- [Truffle](https://trufflesuite.com/)
- [Hardhat](https://hardhat.org/)
- [Foundry](https://github.com/foundry-rs/foundry)
- [Brownie](https://eth-brownie.readthedocs.io/en/stable/)

...it all just works!

For demos of deploying with hardhat see the [Pet Shop](https://github.com/OffchainLabs/arbitrum-tutorials/tree/master/packages/demo-dapp-pet-shop) and [Election](https://github.com/OffchainLabs/arbitrum-tutorials/tree/master/packages/demo-dapp-election) dapp tutorials.

For info on new / different behavior between Arbitrum and Ethereum, see [Differences with Ethereum](arb-specific-things).
