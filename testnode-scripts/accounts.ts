import { ethers, BigNumber } from "ethers";
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
    try {
        if (!name.startsWith("user_")) {
            throw ("")
        }
        let usernum = Number(name.substring(5))
        let userhex = usernum.toString(16)
        for (let index = 0; index < 4; index++) {
            userhex = userhex + userhex
        }
        return new ethers.Wallet("0x" + userhex)
    } catch (error) {
        throw Error("account name must either be funnel, sequencer, validator or user_[number]")
    }
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
