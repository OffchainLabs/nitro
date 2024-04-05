import { execSync } from 'child_process'
import { ContractFactory, Signer, ethers } from 'ethers'
import * as http from 'http'
import { IReader4844, IReader4844__factory } from '../../build/types'
import { JsonRpcProvider } from '@ethersproject/providers'
import { bytecode as Reader4844Bytecode } from '../../out/yul/Reader4844.yul/Reader4844.json'

const wait = async (ms: number) =>
  new Promise(res => {
    setTimeout(res, ms)
  })

export class Toolkit4844 {
  public static DATA_BLOB_HEADER_FLAG = '0x50' // 0x40 | 0x10

  public static postDataToGeth(body: any): Promise<any> {
    return new Promise((resolve, reject) => {
      const options = {
        hostname: '127.0.0.1',
        port: 8545,
        path: '/',
        method: 'POST',
        headers: {
          'Content-Type': 'application/json;charset=UTF-8',
          'Content-Length': Buffer.byteLength(JSON.stringify(body)),
        },
      }

      const req = http.request(options, res => {
        let data = ''

        // Event emitted when a chunk of data is received
        res.on('data', chunk => {
          data += chunk
        })

        // Event emitted when the response is fully received
        res.on('end', () => {
          resolve(JSON.parse(data))
        })
      })

      // Handle any errors
      req.on('error', error => {
        reject(error)
      })

      // Send the POST data
      req.write(JSON.stringify(body))

      // Close the request
      req.end()
    })
  }

  public static async getTx(
    txHash: string
  ): Promise<ethers.providers.TransactionResponse> {
    const body = {
      method: 'eth_getTransactionByHash',
      params: [txHash],
      id: Date.now(),
      jsonrpc: '2.0',
    }
    return (await this.postDataToGeth(body))['result']
  }

  public static async getTxReceipt(
    txHash: string
  ): Promise<ethers.providers.TransactionReceipt> {
    const body = {
      method: 'eth_getTransactionReceipt',
      params: [txHash],
      id: Date.now(),
      jsonrpc: '2.0',
    }
    return (await this.postDataToGeth(body))['result']
  }

  public static async chainId(): Promise<any> {
    const body = {
      method: 'eth_chainId',
      params: [],
      id: Date.now(),
      jsonrpc: '2.0',
    }
    return (await this.postDataToGeth(body))['result']
  }

  public static isReplacementError(err: string) {
    const errRegex =
      /Error while sending transaction: replacement transaction underpriced:/
    const match = err.match(errRegex)
    return Boolean(match)
  }

  public static async waitUntilBlockMined(
    blockNumber: number,
    provider: JsonRpcProvider
  ) {
    while ((await provider.getBlockNumber()) <= blockNumber) {
      await wait(300)
    }
  }

  public static async sendBlobTx(
    privKey: string,
    to: string,
    blobs: string[],
    data: string
  ) {
    const blobStr = blobs.reduce((acc, blob) => acc + ' -b ' + blob, '')
    const blobCommand = `docker run --network=nitro-testnode_default ethpandaops/goomy-blob@sha256:8fd6dfe19bedf43f485f1d5ef3db0a0af569c1a08eacc117d5c5ba43656989f0 blob-sender -p ${privKey} -r http://geth:8545 -t ${to} -d ${data} --gaslimit 1000000${blobStr} 2>&1`
    const res = execSync(blobCommand).toString()
    const txHashRegex = /0x[a-fA-F0-9]{64}/
    const match = res.match(txHashRegex)
    if (match) {
      await wait(10000)
      return match[0]
    } else {
      throw new Error('Error sending blob tx:\n' + res)
    }
  }

  public static async deployReader4844(wallet: Signer): Promise<IReader4844> {
    const contractFactory = new ContractFactory(
      IReader4844__factory.abi,
      Reader4844Bytecode,
      wallet
    )
    const reader4844 = await contractFactory.deploy()
    await reader4844.deployed()

    return IReader4844__factory.connect(reader4844.address, wallet)
  }
}
