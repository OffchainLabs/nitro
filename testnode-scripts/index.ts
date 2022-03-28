import { hideBin } from 'yargs/helpers';
import yargs from 'yargs/yargs';
import { redisReadCommand, redisInitCommand } from './redis'
import { writeConfigCommand } from './config'
import { printAddressCommand } from "./accounts";
import { bridgeFundsCommand, sendL1FundsCommand } from './ethcommands'

async function main() {
    await yargs(hideBin(process.argv))
        .command(bridgeFundsCommand)
        .command(sendL1FundsCommand)
        .command(writeConfigCommand)
        .command(printAddressCommand)
        .command(redisReadCommand)
        .command(redisInitCommand)
        .help()
        .parse()
}

main()
    .then(() => process.exit(0))
    .catch(error => {
        console.error(error)
        process.exit(1)
    })
