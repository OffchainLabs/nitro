import { hideBin } from 'yargs/helpers';
import Yargs from 'yargs/yargs';
import { redisReadCommand, redisInitCommand } from './redis'
import { writeConfigCommand } from './config'
import { printAddressCommand } from "./accounts";
import { bridgeFundsCommand, sendL1FundsCommand, sendL2FundsCommand } from './ethcommands'

async function main() {
    await Yargs(hideBin(process.argv))
        .command(bridgeFundsCommand)
        .command(sendL1FundsCommand)
        .command(sendL2FundsCommand)
        .command(writeConfigCommand)
        .command(printAddressCommand)
        .command(redisReadCommand)
        .command(redisInitCommand)
        .strict()
        .demandCommand(1, 'a command must be specified')
        .help()
        .argv
}

main()
    .then(() => process.exit(0))
    .catch(error => {
        console.error(error)
        process.exit(1)
    })
