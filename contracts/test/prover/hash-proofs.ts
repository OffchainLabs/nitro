import { assert } from 'chai'
import { ethers, run } from 'hardhat'

describe('HashProofHelper', function () {
  it('Should produce valid proofs from full preimages', async function () {
    await run('deploy', { tags: 'HashProofHelper' })

    const hashProofHelper = await ethers.getContract('HashProofHelper')

    for (let i = 0; i < 16; i += 1) {
      const len = Math.floor(Math.random() * 256)
      const data = Math.floor(Math.random() * 256)
      const offset = Math.floor(Math.random() * 256)
      const bytes = Array(len).fill(data)
      const hash = ethers.utils.keccak256(bytes)

      const proofTx = await hashProofHelper.proveWithFullPreimage(bytes, offset)
      const receipt = await proofTx.wait()
      const log = hashProofHelper.interface.parseLog(receipt.logs[0])
      const provenPart = await hashProofHelper.getPreimagePart(hash, offset)

      let dataHex = data.toString(16)
      dataHex = '00'.slice(dataHex.length) + dataHex
      const partLen = Math.min(32, Math.max(0, len - offset))
      const partString = '0x' + dataHex.repeat(partLen)
      assert.equal(log.args['fullHash'], hash)
      assert.equal(log.args['offset'], offset)
      assert.equal(log.args['part'], partString)
      assert.equal(provenPart, partString)
    }
  })

  it('Should produce valid proofs from split preimages', async function () {
    await run('deploy', { tags: 'HashProofHelper' })

    const hashProofHelper = await ethers.getContract('HashProofHelper')

    for (let i = 0; i < 16; i += 1) {
      const len = Math.floor(Math.random() * 1024)
      const data = Math.floor(Math.random() * 256)
      const offset = Math.floor(Math.random() * 256)
      const bytes = Array(len).fill(data)
      const hash = ethers.utils.keccak256(bytes)

      let provenLen = 0
      let provenPart = null
      let log = null
      while (provenPart === null) {
        let nextPartialLen = 136 * (1 + Math.floor(Math.random() * 2))
        if (nextPartialLen > len - provenLen) {
          nextPartialLen = len - provenLen
        }
        const newProvenLen = provenLen + nextPartialLen
        const isFinal = newProvenLen == len
        const proofTx = await hashProofHelper.proveWithSplitPreimage(
          bytes.slice(provenLen, newProvenLen),
          offset,
          isFinal ? 1 : 0
        )
        const receipt = await proofTx.wait()
        if (receipt.logs.length > 0) {
          log = hashProofHelper.interface.parseLog(receipt.logs[0])
          provenPart = await hashProofHelper.getPreimagePart(hash, offset)
        }
        provenLen = newProvenLen
      }

      let dataHex = data.toString(16)
      dataHex = '00'.slice(dataHex.length) + dataHex
      const partLen = Math.min(32, Math.max(0, len - offset))
      const partString = '0x' + dataHex.repeat(partLen)
      assert.isNotNull(log)
      assert.equal(log!.args['fullHash'], hash)
      assert.equal(log!.args['offset'], offset)
      assert.equal(log!.args['part'], partString)
      assert.equal(provenPart, partString)
    }
  })
})
