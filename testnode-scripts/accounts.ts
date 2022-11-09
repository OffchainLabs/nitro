import { ethers } from "ethers";
import * as consts from "./consts";
import * as fs from "fs";
import * as crypto from "crypto";
import { runStress } from "./stress";
const path = require("path");

const accountsToCreate = 3;
let knownaccounts: ethers.Wallet[];

async function createAccounts() {
  for (let i = 0; i < accountsToCreate; i++) {
    const wallet = ethers.Wallet.fromMnemonic(
      consts.l1mnemonic,
      "m/44'/60'/0'/0/" + i
    );
    let walletJSON = await wallet.encrypt(consts.l1passphrase);
    fs.writeFileSync(
      path.join(consts.l1keystore, wallet.address + ".key"),
      walletJSON
    );
  }
}

function possiblyInitKnownAccounts() {
  if (knownaccounts != undefined && knownaccounts.length > 0) {
    return;
  }
  let keyFilenames = fs.readdirSync(consts.l1keystore);
  keyFilenames.sort();

  knownaccounts = keyFilenames.map((filename) => {
    const walletStr = fs
      .readFileSync(path.join(consts.l1keystore, filename))
      .toString();
    return ethers.Wallet.fromEncryptedJsonSync(walletStr, consts.l1passphrase);
  });
}

export function namedAccount(
  name: string,
  threadId?: number | undefined
): ethers.Wallet {
  if (name == "funnel") {
    possiblyInitKnownAccounts();
    return knownaccounts[0];
  }
  if (name == "sequencer") {
    possiblyInitKnownAccounts();
    return knownaccounts[1];
  }
  if (name == "validator") {
    possiblyInitKnownAccounts();
    return knownaccounts[2];
  }
  if (name.startsWith("user_")) {
    return new ethers.Wallet(
      ethers.utils.sha256(ethers.utils.toUtf8Bytes(name))
    );
  }
  if (name.startsWith("threaduser_")) {
    if (threadId == undefined) {
      throw Error("threaduser_ account used but not supported here");
    }
    return new ethers.Wallet(
      ethers.utils.sha256(
        ethers.utils.toUtf8Bytes(
          name.substring(6) + "_thread_" + threadId.toString()
        )
      )
    );
  }
  if (name.startsWith("key_")) {
    return new ethers.Wallet(ethers.utils.hexlify(name.substring(4)));
  }
  throw Error("bad account name: [" + name + "] see general help");
}

export function namedAddress(
  name: string,
  threadId?: number | undefined
): string {
  if (name.startsWith("address_")) {
    return name.substring(8);
  }
  if (name == "random") {
    return "0x" + crypto.randomBytes(20).toString("hex");
  }
  return namedAccount(name, threadId).address;
}

export const namedAccountHelpString =
  "Valid account names:\n" +
  "funnel | sequencer | validator - read from keystore (first 3 keys)\n" +
  "user_[Alphanumeric]            - key will be generated from username\n" +
  "threaduser_[Alphanumeric]      - same as user_[Alphanumeric]_thread_[thread-id]\n" +
  "key_0x[full private key]       - user with specified private key";
"\n" +
  "Valid addresses: any account name, or\n" +
  "address_0x[full eth address]\n" +
  "random";

async function handlePrintAddress(argv: any, threadId: number) {
  console.log(namedAddress(argv.account, threadId));
}

export const printAddressCommand = {
  command: "print-address",
  describe: "prints the requested address",
  builder: {
    account: {
      string: true,
      describe: "address (see general help)",
      default: "funnel",
    },
  },
  handler: async (argv: any) => {
    await runStress(argv, handlePrintAddress);
  },
};

export const writeAccountsCommand = {
  command: "write-accounts",
  describe: "writes wallet files",
  handler: async (argv: any) => {
    await createAccounts();
  },
};
