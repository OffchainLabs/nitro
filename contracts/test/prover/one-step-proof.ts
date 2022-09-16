import { ethers, run, getNamedAccounts } from 'hardhat'
import fs from 'fs'
import assert from 'assert'
import readline from 'readline'

const PARALLEL = 128

async function sendTestMessages() {
  const { deployer } = await getNamedAccounts()
  const inbox = await ethers.getContract('InboxStub', deployer)
  const seqInbox = await ethers.getContract('SequencerInboxStub', deployer)
  const msgRoot = '../arbitrator/prover/test-cases/rust/data/'
  const gasOpts = {
    gasLimit: ethers.utils.hexlify(250000),
    gasPrice: ethers.utils.parseUnits('5', 'gwei'),
  }
  for (let msgNum = 0; msgNum < 2; msgNum++) {
    const path = msgRoot + 'msg' + String(msgNum) + '.bin'
    const buf = fs.readFileSync(path)
    await inbox.sendL2MessageFromOrigin(buf, gasOpts)
    // Don't use the FromOrigin variant as the stub will fail to create a batch posting report
    await seqInbox.addSequencerL2Batch(
      msgNum,
      buf,
      0,
      ethers.constants.AddressZero,
      0,
      0,
      gasOpts
    )
  }
}

describe('OneStepProof', function () {
  const arbProofsRoot = './test/prover/proofs/'
  const specProofsRoot = './test/prover/spec-proofs/'

  before(async function () {
    await run('deploy', { tags: 'OneStepProofEntry' })
    await run('deploy', { tags: 'SequencerInboxStub' })
    await run('deploy', { tags: 'InboxStub' })
    await sendTestMessages()
  })

  const proofs = []
  for (const file of fs.readdirSync(arbProofsRoot)) {
    if (!file.endsWith('.json')) continue
    proofs.push([arbProofsRoot + file, file])
  }
  if (fs.existsSync(specProofsRoot)) {
    for (const file of fs.readdirSync(specProofsRoot)) {
      if (!file.endsWith('.json')) continue
      proofs.push([specProofsRoot + file, file])
    }
  }

  // it('should deploy test harness with ' + proofs.length + ' proofs', function () {});

  for (const [path, file] of proofs) {
    it('Should pass ' + file + ' proofs', async function () {
      const proofs = JSON.parse(fs.readFileSync(path).toString('utf8'))
      const osp = await ethers.getContract('OneStepProofEntry')
      const bridge = await ethers.getContract('BridgeStub')

      const promises = []
      const isdone = []
      for (let i = 0; i < proofs.length; i++) {
        process.stdout.write(
          '\rTesting ' + file + ' proof ' + i + '/' + proofs.length + ' '
        )
        const proof = proofs[i]
        isdone.push(false)
        const inboxLimit = 1000000
        const promise = osp
          .proveOneStep(
            [inboxLimit, bridge.address],
            i,
            [...Buffer.from(proof.before, 'hex')],
            [...Buffer.from(proof.proof, 'hex')]
          )
          .catch((err: any) => {
            console.error('Error executing proof ' + i, err.reason)
            throw err
          })
          .then((after: any) =>
            assert.equal(
              after,
              '0x' + proof.after,
              "After state doesn't match after proof " + i
            )
          )
          .finally((_: any) => {
            isdone[i] = true
          })
        if (promises.length < PARALLEL) {
          promises.push(promise)
        } else {
          const finished: any = await Promise.race(
            promises.map((p, k) => p.then((_: any) => k))
          )
          promises[finished] = promise
        }
      }

      let stillWaiting = []
      do {
        if (promises.length == 0) {
          break
        }
        const finished: any = await Promise.race(
          promises.map((p, k) => p.then((_: any) => k))
        )
        if (finished == promises.length - 1) {
          promises.pop()
        } else {
          promises[finished] = promises.pop()
        }
        stillWaiting = []
        for (let i = 0; i < isdone.length; i++) {
          if (!isdone[i]) {
            stillWaiting.push(i)
          }
        }
        readline.clearLine(process.stdout, 0)
        process.stdout.write(
          '\rTesting ' +
            file +
            ' Waiting for: ' +
            String(stillWaiting.length) +
            '/' +
            String(isdone.length)
        )
        if (stillWaiting.length < 10) {
          process.stdout.write(': ')
          for (let i = 0; i < stillWaiting.length; i++) {
            process.stdout.write(String(stillWaiting[i]) + ',')
          }
        }
      } while (stillWaiting.length > 0)
      await Promise.all(promises)
      process.stdout.write('\r')
    })
  }
})
