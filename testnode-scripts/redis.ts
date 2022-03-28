import { createClient } from '@node-redis/client';
import * as consts from './consts'
import yargs, { Argv } from 'yargs';


async function readRedis(key: string) {
    const redis = createClient({ url: consts.redisUrl })
    await redis.connect()

    const val = await redis.get(key)
    console.log("redis[%s]:%s", key, val)
}

export const redisReadCommand = {
    command: "redis-read",
    describe: "read key",
    builder: (yargs: Argv) => yargs.positional('key', {
        type: 'string',
        describe: 'key to read',
        default: 'coordinator.priorities'
    }),
    handler: (argv: any) => {
        readRedis(argv.key)
    }
}

async function writeRedisPriorities(priorities: number) {
    const redis = createClient({ url: consts.redisUrl })

    let prio_sequencers = "bcd"
    let priostring = ""
    if (priorities == 0) {
        priostring = "ws://sequencer:7546"
    }
    if (priorities > prio_sequencers.length) {
        priorities = prio_sequencers.length
    }
    for (let index = 0; index < priorities; index++) {
        const this_prio = "ws://sequencer_" + prio_sequencers.charAt(index) + ":7546"
        if (index != 0) {
            priostring = priostring + ","
        }
        priostring = priostring + this_prio
    }
    await redis.connect()

    await redis.set("coordinator.priorities", priostring)
}

export const redisInitCommand = {
    command: "redis-init",
    describe: "init redis priorities",
    builder: (yargs: Argv) => yargs.positional('redundancy', {
        type: 'number',
        describe: 'number of servers [0-3]',
        default: 0
    }),
    handler: (argv: any) => {
        writeRedisPriorities(argv.redundancy)
    }
}
