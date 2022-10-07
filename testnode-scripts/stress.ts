
export const stressOptions = {
    times: { number: true, description: 'times to repeat per thread', default: 1 },
    delay: { number: true, description: 'delay between repeats (ms)', default: 0 },
    threads: { number: true, default: 1 },
    threadId: { number: true, description: 'first thread-Id used', default: 0 },
    serial: { boolean: true, description: 'do all actions serially (e.g. when from is identical for all threads)', default: false }
}


async function runThread(argv: any, threadIndex: number, commandHandler: (argv: any, thread: number) => Promise<void>) {
    await commandHandler(argv, threadIndex)
}

export async function runStress(argv: any, commandHandler: (argv: any, thread: number) => Promise<void>) {
    let promiseArray: Array<Promise<void>>
    promiseArray = []
    for (let threadIndex = 0; threadIndex < argv.threads; threadIndex++) {
        const threadPromise = runThread(argv, threadIndex + argv.threadId, commandHandler)
        if (argv.serial) {
            await threadPromise
        } else {
            promiseArray.push(threadPromise)
        }
    }
    await Promise.all(promiseArray)
        .catch(error => {
            console.error(error)
            process.exit(1)
        })
}
