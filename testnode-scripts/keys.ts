import { ethers } from "ethers";
import { Argv } from 'yargs';
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

export function namedAccount(name: "funnel" | "sequencer" | "validator"): ethers.Wallet {
    possiblyInitKnownAccounts()

    if (name == "funnel") {
        return knownaccounts[0]
    }
    if (name == "sequencer") {
        return knownaccounts[1]
    }
    return knownaccounts[2]
}

export const accountChooser = {
    choices: ["funnel", "sequencer", "validator"] as const,
    default: "funnel"
}

export const printAddressCommand = {
    command: "print-address",
    describe: "sends funds from l1 to l2",
    builder: (yargs: Argv) => yargs.options({
        account: accountChooser,
    }),
    handler: (argv: any) => {
        console.log(namedAccount(argv.account).address)
    }
}
