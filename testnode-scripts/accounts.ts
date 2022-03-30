import { ethers } from "ethers";
import * as consts from './consts'
import * as fs from 'fs';
const path = require("path");

let knownaccounts: ethers.Wallet[]

function possiblyInitKnownAccounts() {
    if (knownaccounts != undefined && knownaccounts.length > 0) {
        return;
    }
    let keyFilenames = fs.readdirSync(consts.l1keystore)
    keyFilenames.sort()

    knownaccounts = keyFilenames.map((filename) => {
        return ethers.Wallet.fromEncryptedJsonSync(fs.readFileSync(path.join(consts.l1keystore, filename)).toString(), consts.l1passphrase)
    })
}

export function namedAccount(name: string): ethers.Wallet {
    possiblyInitKnownAccounts()

    if (name == "funnel") {
        return knownaccounts[0]
    }
    if (name == "sequencer") {
        return knownaccounts[1]
    }
    if (name == "validator") {
        return knownaccounts[2]
    }
    if (name.startsWith("user_")) {
        return new ethers.Wallet(ethers.utils.sha256(ethers.utils.toUtf8Bytes(name)))
    }
    throw Error("account name must either be funnel, sequencer, validator or user_[number]")
}

export function namedAddress(name: string): string {
    if (name.startsWith("address_")) {
        return name.substring(8)
    }
    return namedAccount(name).address
}

export const printAddressCommand = {
    command: "print-address",
    describe: "sends funds from l1 to l2",
    builder: {
        account: { string: true, describe: "account name", default: "funnel" },
    },
    handler: (argv: any) => {
        console.log(namedAccount(argv.account).address)
    }
}
