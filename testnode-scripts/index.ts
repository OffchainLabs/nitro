import { hideBin } from "yargs/helpers";
import Yargs from "yargs/yargs";
import { stressOptions } from "./stress";
import { redisReadCommand, redisInitCommand } from "./redis";
import { writeConfigCommand, writeGethGenesisCommand, writePrysmCommand } from "./config";
import {
  printAddressCommand,
  namedAccountHelpString,
  writeAccountsCommand,
} from "./accounts";
import {
  bridgeFundsCommand,
  sendL1Command,
  sendL2Command,
  sendRPCCommand,
} from "./ethcommands";

async function main() {
  await Yargs(hideBin(process.argv))
    .options({
      redisUrl: { string: true, default: "redis://redis:6379" },
      l1url: { string: true, default: "ws://geth:8546" },
      l2url: { string: true, default: "ws://sequencer:8548" },
      validationNodeUrl: { string: true, default: "ws://validation_node:8549" },
    })
    .options(stressOptions)
    .command(bridgeFundsCommand)
    .command(sendL1Command)
    .command(sendL2Command)
    .command(sendRPCCommand)
    .command(writeConfigCommand)
    .command(writeGethGenesisCommand)
    .command(writePrysmCommand)
    .command(writeAccountsCommand)
    .command(printAddressCommand)
    .command(redisReadCommand)
    .command(redisInitCommand)
    .strict()
    .demandCommand(1, "a command must be specified")
    .epilogue(namedAccountHelpString)
    .help().argv;
}

main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.error(error);
    process.exit(1);
  });
