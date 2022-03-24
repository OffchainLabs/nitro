import { ethers, BigNumber } from "ethers";
import * as fs from 'fs';
import { boolean } from "yargs";
import { hideBin } from 'yargs/helpers';
import yargs from 'yargs/yargs';
const path = require("path");

const l1keystore = "/l1keystore"
const l1passphrase = "passphrase"
const configpath = "/config"

async function createSendTransaction(provider: ethers.providers.Provider, from: ethers.Wallet, to: string, value: ethers.BigNumberish, data: ethers.BytesLike): Promise<ethers.providers.TransactionResponse> {
    const nonce = await provider.getTransactionCount(from.address, "latest");
    const chainId = (await provider.getNetwork()).chainId

    var transactionRequest: ethers.providers.TransactionRequest = {
        type: 2,
        from: from.address,
        to: to,
        value: value,
        data: data,
        nonce: nonce,
        chainId: chainId,
    }
    const gasEstimate = await provider.estimateGas(transactionRequest)

    let feeData = await provider.getFeeData();
    if (feeData.maxPriorityFeePerGas == null || feeData.maxFeePerGas == null) {
        throw Error("bad L1 fee data")
    }
    transactionRequest.gasLimit = BigNumber.from(Math.ceil(gasEstimate.toNumber() * 1.2))
    transactionRequest.maxPriorityFeePerGas = BigNumber.from(Math.ceil(feeData.maxPriorityFeePerGas.toNumber() * 1.2)), // Recommended maxPriorityFeePerGas
    transactionRequest.maxFeePerGas = BigNumber.from(Math.ceil(feeData.maxFeePerGas.toNumber() * 1.2))

    const signedTx = await from.signTransaction(transactionRequest)

    return provider.sendTransaction(signedTx)
}


function writeConfigs(sequenceraddress: string, validatoraddress: string) {
    const baseConfig = {
        "l1": {
            "deployment": "/config/deployment.json",
            "wallet": {
                "account": "",
                "password": l1passphrase,
                "pathname": l1keystore,
            },
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
            }
        },
        "persistent": {
            "data": "/data"
        },
        "ws": {
            "addr": "0.0.0.0"
        },
        "http": {
            "addr": "0.0.0.0"
        },
    }
    const baseConfJSON = JSON.stringify(baseConfig)

    var validatorConfig = JSON.parse(baseConfJSON)
    validatorConfig.l1.wallet.account = validatoraddress
    validatorConfig.node.validator.enable = true
    var validconfJSON = JSON.stringify(validatorConfig)
    fs.writeFileSync(path.join(configpath, "validator_config.json"), validconfJSON)

    var unsafeStakerConfig = JSON.parse(validconfJSON)
    unsafeStakerConfig.node.validator.dangerous["without-block-validator"] = true
    fs.writeFileSync(path.join(configpath, "unsafe_staker_config.json"), JSON.stringify(unsafeStakerConfig))

    var sequencerConfig = JSON.parse(baseConfJSON)
    sequencerConfig.l1.wallet.account = sequenceraddress
    sequencerConfig.node.sequencer.enable = true
    fs.writeFileSync(path.join(configpath, "sequencer_config.json"), JSON.stringify(sequencerConfig))
}

async function bridgeFunds(provider: ethers.providers.Provider, from: ethers.Wallet): Promise<ethers.providers.TransactionResponse> {
    const deploydata = JSON.parse(fs.readFileSync(path.join(configpath, "deployment.json")).toString())
    return createSendTransaction(provider, from, deploydata.Inbox, "0x82f79cd90000", "0x0f4d14e9000000000000000000000000000000000000000000000000000082f79cd90000")
}

async function main() {
    const argv = yargs(hideBin(process.argv)).options({
        writeconfig: {type: 'boolean', describe: 'write config'},
        bridgefunds: {type: 'boolean', describe: 'bridge funds'},
        ethamount: {type: 'string', describe:'amount to transfer (in eth)', default: "10"},
        l1account: {choices: ["funnel", "sequencer", "validator"] as const, default: "funnel"},
        fund: {type: 'boolean', describe: 'send funds from funnel'},
        printaddress: {type: 'boolean', describe: 'print address'}
    }).help().parseSync()

    var keyFilenames = fs.readdirSync(l1keystore)
    keyFilenames.sort()

    let chosenAccount = 0
    if (argv.l1account == "sequencer") {
        chosenAccount = 1
    }
    if (argv.l1account == "validator") {
        chosenAccount = 2
    }

    let accounts = keyFilenames.map((filename) => {
        return ethers.Wallet.fromEncryptedJsonSync(fs.readFileSync(path.join(l1keystore,filename)).toString(), l1passphrase)
    })

    var provider = new ethers.providers.WebSocketProvider("ws://geth:8546")

    if (argv.fund) {
        var response = await createSendTransaction(provider, accounts[0], accounts[chosenAccount].address, ethers.utils.parseEther(argv.ethamount), new Uint8Array())
        console.log("sent " + argv.l1account + " funding")
        console.log(response)
    }

    if (argv.writeconfig) {
        writeConfigs(accounts[1].address, accounts[2].address)
        console.log("config files written")
    }

    if (argv.bridgefunds) {
        var response = await bridgeFunds(provider, accounts[chosenAccount])
        console.log("bridged funds")
        console.log(response)
    }

    if (argv.printaddress) {
        console.log(accounts[chosenAccount].address)
    }
    provider.destroy()
}

main();
