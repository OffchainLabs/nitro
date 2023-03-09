/*
 * Copyright 2021, Offchain Labs, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
/* eslint-env node */
'use strict'

import { JsonRpcProvider } from '@ethersproject/providers'
import dotenv from 'dotenv'
import { L1Network, L2Network, addCustomNetwork } from '@arbitrum/sdk'
import { execSync } from 'child_process'
import { Bridge__factory } from '@arbitrum/sdk/dist/lib/abi/factories/Bridge__factory'
import { RollupAdminLogic__factory } from '@arbitrum/sdk/dist/lib/abi/factories/RollupAdminLogic__factory'

dotenv.config()

export const config = {
  arbUrl: process.env['ARB_URL'] as string,
  ethUrl: process.env['ETH_URL'] as string,

  arbKey: process.env['ARB_KEY'] as string,
  ethKey: process.env['ETH_KEY'] as string,
}

export const getCustomNetworks = async (
  l1Url: string,
  l2Url: string
): Promise<{
  l1Network: L1Network
  l2Network: Omit<L2Network, 'tokenBridge'> & { nativeToken: string }
}> => {
  const l1Provider = new JsonRpcProvider(l1Url)
  const l2Provider = new JsonRpcProvider(l2Url)
  let deploymentData: string
  try {
    deploymentData = execSync(
      'docker exec nitro_sequencer_1 cat /config/deployment.json'
    ).toString()
  } catch (e) {
    deploymentData = execSync(
      'docker exec nitro-sequencer-1 cat /config/deployment.json'
    ).toString()
  }
  const parsedDeploymentData = JSON.parse(deploymentData) as {
    bridge: string
    inbox: string
    ['sequencer-inbox']: string
    rollup: string
    ['native-erc20-token']: string
  }

  const rollup = RollupAdminLogic__factory.connect(
    parsedDeploymentData.rollup,
    l1Provider
  )
  const confirmPeriodBlocks = await rollup.confirmPeriodBlocks()

  const bridge = Bridge__factory.connect(
    parsedDeploymentData.bridge,
    l1Provider
  )
  const outboxAddr = await bridge.allowedOutboxList(0)

  const l1NetworkInfo = await l1Provider.getNetwork()
  const l2NetworkInfo = await l2Provider.getNetwork()

  const l1Network: L1Network = {
    blockTime: 10,
    chainID: l1NetworkInfo.chainId,
    explorerUrl: '',
    isCustom: true,
    name: 'EthLocal',
    partnerChainIDs: [l2NetworkInfo.chainId],
    isArbitrum: false,
  }

  const l2Network: Omit<L2Network, 'tokenBridge'> & { nativeToken: string } = {
    chainID: l2NetworkInfo.chainId,
    confirmPeriodBlocks: confirmPeriodBlocks.toNumber(),
    ethBridge: {
      bridge: parsedDeploymentData.bridge,
      inbox: parsedDeploymentData.inbox,
      outbox: outboxAddr,
      rollup: parsedDeploymentData.rollup,
      sequencerInbox: parsedDeploymentData['sequencer-inbox'],
    },
    nativeToken: parsedDeploymentData['native-erc20-token'],
    explorerUrl: '',
    isArbitrum: true,
    isCustom: true,
    name: 'ArbLocal',
    partnerChainID: l1NetworkInfo.chainId,
    retryableLifetimeSeconds: 7 * 24 * 60 * 60,
    nitroGenesisBlock: 0,
    depositTimeout: 900000,
  }
  return {
    l1Network,
    l2Network,
  }
}

export const setupNetworks = async (l1Url: string, l2Url: string) => {
  const { l1Network, l2Network: coreL2Network } = await getCustomNetworks(
    l1Url,
    l2Url
  )
  const l2Network: L2Network & { nativeToken: string } = {
    ...coreL2Network,
    tokenBridge: {
      l1CustomGateway: '',
      l1ERC20Gateway: '',
      l1GatewayRouter: '',
      l1MultiCall: '',
      l1ProxyAdmin: '',
      l1Weth: '',
      l1WethGateway: '',

      l2CustomGateway: '',
      l2ERC20Gateway: '',
      l2GatewayRouter: '',
      l2Multicall: '',
      l2ProxyAdmin: '',
      l2Weth: '',
      l2WethGateway: '',
    },
  }

  addCustomNetwork({
    customL1Network: l1Network,
    customL2Network: l2Network,
  })

  return {
    l1Network,
    l2Network,
  }
}
