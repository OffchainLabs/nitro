import { ethers, BigNumber } from "ethers";
import * as fs from 'fs';
import { hideBin } from 'yargs/helpers';
const yargs = require("yargs")
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
    const argv = yargs(hideBin(process.argv)).argv

    var keyFilenames = fs.readdirSync(l1keystore)
    keyFilenames.sort()

    const sequencerAccount = ethers.Wallet.fromEncryptedJsonSync(fs.readFileSync(path.join(l1keystore, keyFilenames[0])).toString(), l1passphrase)
    const validatorAccount = ethers.Wallet.fromEncryptedJsonSync(fs.readFileSync(path.join(l1keystore, keyFilenames[1])).toString(), l1passphrase)

    var provider = new ethers.providers.WebSocketProvider("ws://geth:8546")

    if (argv.fundvalidator) {
        var response = await createSendTransaction(provider, sequencerAccount, validatorAccount.address, ethers.utils.parseEther("10"), new Uint8Array())
        console.log("sent validator funding")
        console.log(response)
    }

    if (argv.writeconfig) {
        writeConfigs(sequencerAccount.address, validatorAccount.address)
        console.log("config files written")
    }

    if (argv.bridgefunds) {
        var response = await bridgeFunds(provider, sequencerAccount)
        console.log("bridged funds")
        console.log(response)
    }

    if (argv.printsequenceraddress) {
        console.log(sequencerAccount.address)
    }
    provider.destroy()
}

main();
