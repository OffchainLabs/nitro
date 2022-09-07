import * as fs from 'fs';
import * as consts from './consts'
import { namedAccount } from './accounts'

const path = require("path");

function writeConfigs(argv: any) {
    const deployment = JSON.parse(fs.readFileSync(path.join(consts.configpath, "deployment.json")).toString('utf-8'));
    const baseConfig = {
        "l1": {
            "rollup": deployment,
            "url": argv.l1url,
            "wallet": {
                "account": "",
                "password": consts.l1passphrase,
                "pathname": consts.l1keystore,
            },
        },
        "l2": {
            "chain-id": 412346,
            "dev-wallet" : {
                "private-key": "e887f7d17d07cc7b8004053fb8826f6657084e88904bb61590e498ca04704cf2"
            }
        },
        "node": {
            "archive": true,
            "forwarding-target": "null",
            "validator": {
                "dangerous": {
                    "without-block-validator": false
                },
                "disable-challenge": false,
                "enable": false,
                "staker-interval": "10s",
                "strategy": "MakeNodes",
                "target-machine-count": 4,
            },
            "sequencer": {
                "enable": false
            },
            "delayed-sequencer": {
                "enable": false
            },
            "seq-coordinator": {
                "enable": false,
                "redis-url": argv.redisUrl,
                "lockout-duration": "30s",
                "lockout-spare": "1s",
                "my-url": "",
                "retry-interval": "0.5s",
                "seq-num-duration": "24h0m0s",
                "update-interval": "3s",
                "signer" : {
                    "signing-key": "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
                }
            },
            "batch-poster": {
                "enable": false,
                "redis-lock": {
                    "redis-url": argv.redisUrl,
                    "key": "batchPosterLock",
                },
                "max-interval": "30s",
            }
        },
        "init": {
            "dev-init": "true"
        },
        "persistent": {
	        "chain": "local"
        },
        "ws": {
            "addr": "0.0.0.0"
        },
        "http": {
            "addr": "0.0.0.0",
            "vhosts": "*",
            "corsdomain": "*"
        },
    }
    const baseConfJSON = JSON.stringify(baseConfig)

    let validatorConfig = JSON.parse(baseConfJSON)
    validatorConfig.l1.wallet.account = namedAccount("validator").address
    validatorConfig.node.validator.enable = true
    let validconfJSON = JSON.stringify(validatorConfig)
    fs.writeFileSync(path.join(consts.configpath, "validator_config.json"), validconfJSON)

    let unsafeStakerConfig = JSON.parse(validconfJSON)
    unsafeStakerConfig.node.validator.dangerous["without-block-validator"] = true
    fs.writeFileSync(path.join(consts.configpath, "unsafe_staker_config.json"), JSON.stringify(unsafeStakerConfig))

    let sequencerConfig = JSON.parse(baseConfJSON)
    sequencerConfig.node.sequencer.enable = true
    sequencerConfig.node["seq-coordinator"].enable = true
    sequencerConfig.node["delayed-sequencer"].enable = true
    fs.writeFileSync(path.join(consts.configpath, "sequencer_config.json"), JSON.stringify(sequencerConfig))

    let posterConfig = JSON.parse(baseConfJSON)
    posterConfig.l1.wallet.account = namedAccount("sequencer").address
    posterConfig.node["seq-coordinator"].enable = true
    posterConfig.node["batch-poster"].enable = true
    fs.writeFileSync(path.join(consts.configpath, "poster_config.json"), JSON.stringify(posterConfig))
}

export const writeConfigCommand = {
    command: "write-config",
    describe: "writes config files",
    handler: (argv: any) => {
        writeConfigs(argv)
    }
}

