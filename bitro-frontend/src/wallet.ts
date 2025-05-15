import { ethers } from 'ethers';

export async function connectWallet(): Promise<ethers.providers.Web3Provider> {
  if (!window.ethereum) throw new Error('MetaMask not found');

  const provider = new ethers.providers.Web3Provider(window.ethereum);
  await provider.send('eth_requestAccounts', []);
  return provider;
}
