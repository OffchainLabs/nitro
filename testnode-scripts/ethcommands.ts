import yargs, { Argv } from 'yargs';
import { ethers, BigNumber } from "ethers";
import * as consts from './consts'
import { accountChooser, namedAccount } from './accounts'
import * as fs from 'fs';
const path = require("path");

export async function createSendTransaction(provider: ethers.providers.Provider, from: ethers.Wallet, to: string, value: ethers.BigNumberish, data: ethers.BytesLike): Promise<ethers.providers.TransactionResponse> {
    const nonce = await provider.getTransactionCount(from.address, "latest");
    const chainId = (await provider.getNetwork()).chainId

    let transactionRequest: ethers.providers.TransactionRequest = {
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
    transactionRequest.maxPriorityFeePerGas = BigNumber.from(Math.ceil(feeData.maxPriorityFeePerGas.toNumber() * 1.2)) // Recommended maxPriorityFeePerGas
    transactionRequest.maxFeePerGas = BigNumber.from(Math.ceil(feeData.maxFeePerGas.toNumber() * 1.2))

    const signedTx = await from.signTransaction(transactionRequest)

    return provider.sendTransaction(signedTx)
}

async function bridgeFunds(provider: ethers.providers.Provider, from: ethers.Wallet, ethamount: string): Promise<ethers.providers.TransactionResponse> {
    const deploydata = JSON.parse(fs.readFileSync(path.join(consts.configpath, "deployment.json")).toString())
    return createSendTransaction(provider, from, deploydata.Inbox, ethers.utils.parseEther(ethamount), "0x0f4d14e9000000000000000000000000000000000000000000000000000082f79cd90000")
}

export const bridgeFundsCommand = {
    command: "bridge-funds",
    describe: "sends funds from l1 to l2",
    builder: (yargs: Argv) => yargs.options({
        ethamount: { type: 'string', describe: 'amount to transfer (in eth)', default: "10" },
        account: accountChooser,
    }),
    handler: async (argv: any) => {
        let provider = new ethers.providers.WebSocketProvider(consts.l1url)

        let response = await bridgeFunds(provider, namedAccount(argv.account), argv.ethamount)

        console.log("bridged funds")
        console.log(response)

        provider.destroy()
    }
}

export const sendL1FundsCommand = {
    command: "send-l1",
    describe: "sends funds between l1 accounts",
    builder: (yargs: Argv) => yargs.options({
        ethamount: { type: 'string', describe: 'amount to transfer (in eth)', default: "10" },
        from: accountChooser,
        to: accountChooser,
    }),
    handler: async (argv: any) => {
        let provider = new ethers.providers.WebSocketProvider(consts.l1url)

        let response = await createSendTransaction(provider, namedAccount(argv.from), namedAccount(argv.to).address, ethers.utils.parseEther(argv.ethamount), new Uint8Array())

        console.log("sent funds")
        console.log(response)

        provider.destroy()
    }
}
