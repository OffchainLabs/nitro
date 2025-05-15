import { ethers } from 'ethers';
import { connectWallet } from './wallet';
import { IInbox__factory } from '@arbitrum/sdk/dist/lib/abi/factories/IInbox__factory';

const INBOX_ADDRESS = '0xb811fA75EA2952112c12929f6d11A99C7726f67E';

export async function sendToDelayedInbox(l3TargetAddress: string, messageData: string) {
  const provider = await connectWallet();
  const signer = provider.getSigner();
  const sender = await signer.getAddress();

  const encoded = ethers.utils.defaultAbiCoder.encode(
    ['address', 'bytes'],
    [l3TargetAddress, ethers.utils.toUtf8Bytes(messageData)]
  );

  // Connect to the L2 inbox contract
  const inboxContract = IInbox__factory.connect(INBOX_ADDRESS, signer);

  // These are hardcoded for now — in production, you'd estimate them
  const callValue = ethers.utils.parseEther('0.01');
  const maxSubmissionCost = ethers.utils.parseEther('0.01');
  const gasLimit = 200000;
  const maxFeePerGas = ethers.utils.parseUnits('2', 'gwei');

  const tx = await inboxContract.sendL2Message(
    encoded,            // messageData
  )

  console.log(`TX hash: ${tx.hash}`);
  await tx.wait();
  console.log('Message successfully sent to Bitro L3.');
}
