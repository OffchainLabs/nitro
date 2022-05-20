# Anytrust Goerli Devnet Nitro Node Instructions
The example configuration in this document will allow you to set up an Anytrust Goerli Devnet Nitro node. An Anytrust node runs the same software as a Nitro node, with some additional configuration options supplied. This document assumes you already have a non-Anytrust Nitro deployment configuration that you can adapt to supply the Anytrust related options.

If you do not have Nitro tesnet deployment set up yet, there is non-Anytrust Nitro node setup documentation here: 
* https://developer.arbitrum.io/docs/running_nitro_node
* https://developer.arbitrum.io/docs/public_nitro_devnet

#### Image:
`offchainlabs/nitro-node:v2.0.0-alpha.4`

#### Anytrust-specific node configuration

Options can be supplied to the node entirely on the command line, in a JSON file using the --conf.file flag (eg `--conf.file /config/anytrust-config.json`), or a mix of both. For readability the example config required for an Anytrust node is shown in JSON format below. If the way your installation is configured requires command line options only, to turn any of these entries into a command-line option, trace the json path from outer to inner, eg to specify the l1 rollup bridge contract address, use `--l1.rollup.bridge 0xd1fed339fb57b317dbf3d765310159bf9f614b8c`

As we are in the early stages of the testnet this configuration uses only 2 Offchain Labs hosted DAS endpoints for retrieving the stored batch data. We will provide new configurations for the --node.data-availability.XXX entries as more endpoints become available and the software evolves.

```
{
    "l1":
    {
        "chain-id": 5,
        "url": "wss://goerli.geth.arbitrum-internal.io/ws",
        "rollup":
        {
            "bridge": "0xd1fed339fb57b317dbf3d765310159bf9f614b8c",
            "inbox": "0xa329b51eac558310cc2d6d245c02bfaa85284af0",
            "sequencer-inbox": "0xd5cbd94954d2a694c7ab797d87bf0fb1d49192bf",
            "rollup": "0xd8998fa65d21a64dc8fe9db47205ccd61928d923",
            "validator-utils": "0x5a1cb900922bf0e30fb7938675a50cd8a05fcd14",
            "validator-wallet-creator": "0xef9264d9742775f16b8b1626f126ec52b43e03eb",
            "deployed-at": 6878489
        }
    },
    "l2":
    {
        "chain-id": 421702
    },
    "node":
    {
        "archive": true,
        "feed":
        {
            "input":
            {
                "url": "wss://anytrust-devnet.arbitrum.io/feed"
            }
        },
        "forwarding-target": "https://anytrust-devnet.arbitrum.io/rpc",
        "data-availability":
        {
            "mode": "aggregator",
            "aggregator":
            {
                "assumed-honest": 2,
                "backends": "[{\"url\": \"http://anytrust-devnet.arbitrum.io/das-0\", \"pubkey\": \"YBEh8tfT5JaNSfYBp3tXOqXakSmEhBefTjOGzOn+nv805CzMrea0cKTaeiOxhxsSWQ8MMv71FbCJasbVMq5S3BqKxeWkscyGiVS2fd4nko5XnFEgrVsDBOp1hVatjlnR4RjNG8Y4usla93/H1NKarn/PDHOOPueqAKIMsF8qlnC5VIqVuE8dv4cBOAm6lirkmQKoJqhq3+urcQKwyrcqPfLUmT6nxUdgeq9qKOg7cCY40Ag7I0UA4hZmrqm+1lRlhxmdsVjbex6q16w5gN5rbVw/rjD6UQnIwTsGVk4c74CEYjzxkg1+tiX3whMgEJh6hRnjig5SfizYWgjbssu3rzYSFdDNiVa+rH1ufPF0KxESAUpzB9i+0J9rtDD3LnK1lw==\", \"signermask\": 1 }, {\"url\": \"http://anytrust-devnet.arbitrum.io/das-1\", \"pubkey\": \"YA9iXnZp+n92i63h8MaVCIX84TpwWTOBwPAQlXLlyLP8fAOqbRrf5pkMvZBRMxFPeglCk5DprBJUc4GrqLj2R7APO27QjlAb2BVd85wu8Hxwpkwd2tpkW/OwfGBMe3GJWhDwxmscShUjwpZ38sVQlT/iBy0W6O/lVu4ukwY2D56hxQxwmNcxdMMZGWScsO6AzxekxPpfWBHOnrY1GXvO53yWhiMkAQrsHmmJ0OJga180KgTTDoiw5DL78UTJfaCFdhLbjjmpCApPTGagGs5LAGhNxWczWxt7ClahsuO37ktpMVMzCE7b5V7OJfj+QXVTrAGuJ6e+yIyzT7INmkKDnGSsChfNOAxVan09cZAC/VsCgKR28UuBFQ3g21upIyAFdA==\", \"signermask\": 2}]",
                "sequencer-inbox-address": "0xd5cbd94954d2a694c7ab797d87bf0fb1d49192bf"
            }
        }
    }
}
```
