const Web3 = require('web3')
const fs = require('fs')
const yargs = require('yargs')
const { hideBin } = require('yargs/helpers')
const path = require('path');


async function setupValidator(l1url: string, keystorePath: string, configpath: string, fund: boolean, writeconf: boolean) {
    const web3 = new Web3(l1url)

    var keyFilenames = fs.readdirSync(keystorePath)
    keyFilenames.sort()

    const encKey = JSON.parse(fs.readFileSync(path.join(keystorePath, keyFilenames[0])))
    const decryptedFaucet = web3.eth.accounts.decrypt(encKey, "passphrase")
    const validatorAddress = JSON.parse(fs.readFileSync(path.join(keystorePath, keyFilenames[1]))).address

    if (fund) {
        const valueToSend = web3.utils.toHex(web3.utils.toWei("10", 'ether'))
        const nonce = await web3.eth.getTransactionCount(decryptedFaucet.address, 'latest'); // nonce starts counting from 0
        const gasPrice = await web3.eth.getGasPrice()
        const gasEstimate = await web3.eth.estimateGas({
            from: decryptedFaucet.address,
            to: validatorAddress,
            value: valueToSend,
            nonce: nonce
        })
        const transaction = {
            'to': validatorAddress, // faucet address to return eth
            'value': valueToSend,
            'gas': gasPrice * 2,
            'gasLimit': gasEstimate * 2,
            'nonce': nonce,
            // optional data field to send message or execute smart contract
        };

        const signedTx = await web3.eth.accounts.signTransaction(transaction, decryptedFaucet.privateKey);

        const receipt = await web3.eth.sendSignedTransaction(signedTx.rawTransaction);

        console.log(receipt)
    }

    if (writeconf) {
        const baseConfig = {
            "l1": {
                "deployment": "/config/deployment.json",
                "wallet": {
                    "account": "",
                    "password": "passphrase",
                    "pathname": keystorePath,
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
        validatorConfig.l1.wallet.account = validatorAddress
        validatorConfig.node.validator.enable = true
        var validconfJSON = JSON.stringify(validatorConfig)
        fs.writeFileSync(path.join(configpath, "validator_config.json"), validconfJSON)

        var unsafeStakerConfig = JSON.parse(validconfJSON)
        unsafeStakerConfig.node.validator.dangerous["without-block-validator"] = true
        fs.writeFileSync(path.join(configpath, "unsafe_staker_config.json"), JSON.stringify(unsafeStakerConfig))

        var sequencerConfig = JSON.parse(baseConfJSON)
        sequencerConfig.l1.wallet.account = decryptedFaucet.address
        sequencerConfig.node.sequencer.enable = true
        fs.writeFileSync(path.join(configpath, "sequencer_config.json"), JSON.stringify(sequencerConfig))
    }

    web3.currentProvider.connection.close()
}

async function main() {
    const argv = yargs(hideBin(process.argv)).argv

    console.log(argv)

    await setupValidator(argv.l1url, argv.l1keystore, argv.config, argv.fundvalidator, argv.writeconf)
}

main();
