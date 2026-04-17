//! Shared test helpers for e2e block executor tests.

use std::sync::Arc;

use alloy_consensus::{
    crypto::secp256k1::sign_message,
    transaction::{Recovered, SignerRecoverable},
    EthereumTxEnvelope, SignableTransaction, TxEip1559, TxEip2930, TxLegacy,
};
use alloy_primitives::{address, keccak256, Address, B256, Bytes, TxKind, U256};
use arb_evm::config::ArbEvmConfig;
use arb_primitives::ArbTransactionSigned;
use arb_test_utils::{ArbosHarness, EmptyDb};
use reth_chainspec::ChainSpec;
use reth_evm::EvmEnv;
use revm::{
    context::{BlockEnv, CfgEnv},
    database::{states::account_status::AccountStatus, PlainAccount, State},
    primitives::hardfork::SpecId,
    state::AccountInfo,
};

pub const CHAIN_ID: u64 = 421614;
pub const ONE_GWEI: u128 = 1_000_000_000;
pub const ONE_ETH: u128 = 1_000_000_000_000_000_000;

/// 32-byte secret keys derived for our canonical test accounts.
pub const KEYS: [[u8; 32]; 5] = [
    {
        let mut k = [0u8; 32];
        k[31] = 1;
        k
    },
    {
        let mut k = [0u8; 32];
        k[31] = 2;
        k
    },
    {
        let mut k = [0u8; 32];
        k[31] = 3;
        k
    },
    {
        let mut k = [0u8; 32];
        k[31] = 4;
        k
    },
    {
        let mut k = [0u8; 32];
        k[31] = 5;
        k
    },
];

pub fn derive_address(sk_bytes: [u8; 32]) -> Address {
    use k256::ecdsa::SigningKey;
    let sk = SigningKey::from_slice(&sk_bytes).expect("valid sk");
    let vk = *sk.verifying_key();
    let encoded = vk.to_encoded_point(false);
    let pubkey_bytes = &encoded.as_bytes()[1..];
    let hash = keccak256(pubkey_bytes);
    Address::from_slice(&hash[12..])
}

pub fn alice_key() -> [u8; 32] {
    KEYS[0]
}
pub fn bob_key() -> [u8; 32] {
    KEYS[1]
}
pub fn charlie_key() -> [u8; 32] {
    KEYS[2]
}

pub fn alice() -> Address {
    derive_address(alice_key())
}
pub fn bob() -> Address {
    derive_address(bob_key())
}
pub fn charlie() -> Address {
    derive_address(charlie_key())
}

pub fn fund_account(state: &mut State<EmptyDb>, addr: Address, balance: U256) {
    let _ = state.load_cache_account(addr);
    if let Some(cached) = state.cache.accounts.get_mut(&addr) {
        cached.account = Some(PlainAccount {
            info: AccountInfo {
                balance,
                nonce: 0,
                code_hash: keccak256([]),
                code: None,
                account_id: None,
            },
            storage: Default::default(),
        });
        cached.status = AccountStatus::InMemoryChange;
    }
}

pub fn deploy_contract(
    state: &mut State<EmptyDb>,
    addr: Address,
    runtime_code: Vec<u8>,
    balance: U256,
) {
    use revm::state::Bytecode;
    let _ = state.load_cache_account(addr);
    if let Some(cached) = state.cache.accounts.get_mut(&addr) {
        let bytecode = Bytecode::new_raw(runtime_code.into());
        let code_hash = bytecode.hash_slow();
        cached.account = Some(PlainAccount {
            info: AccountInfo {
                balance,
                nonce: 1,
                code_hash,
                code: Some(bytecode),
                account_id: None,
            },
            storage: Default::default(),
        });
        cached.status = AccountStatus::InMemoryChange;
    }
}

pub fn balance_of(state: &mut State<EmptyDb>, addr: Address) -> U256 {
    state
        .cache
        .accounts
        .get(&addr)
        .and_then(|a| a.account.as_ref())
        .map(|a| a.info.balance)
        .unwrap_or(U256::ZERO)
}

pub fn nonce_of(state: &mut State<EmptyDb>, addr: Address) -> u64 {
    state
        .cache
        .accounts
        .get(&addr)
        .and_then(|a| a.account.as_ref())
        .map(|a| a.info.nonce)
        .unwrap_or(0)
}

pub fn sign_legacy(
    chain_id: u64,
    nonce: u64,
    gas_price: u128,
    gas_limit: u64,
    to: TxKind,
    value: U256,
    input: Bytes,
    sk: [u8; 32],
) -> ArbTransactionSigned {
    let tx = TxLegacy {
        chain_id: Some(chain_id),
        nonce,
        gas_price,
        gas_limit,
        to,
        value,
        input,
    };
    let sig_hash = tx.signature_hash();
    let sig = sign_message(B256::from(sk), sig_hash).expect("sign");
    let signed = tx.into_signed(sig);
    ArbTransactionSigned::from_envelope(EthereumTxEnvelope::Legacy(signed))
}

pub fn sign_1559(
    chain_id: u64,
    nonce: u64,
    max_fee: u128,
    max_priority: u128,
    gas_limit: u64,
    to: TxKind,
    value: U256,
    input: Bytes,
    sk: [u8; 32],
) -> ArbTransactionSigned {
    let tx = TxEip1559 {
        chain_id,
        nonce,
        gas_limit,
        max_fee_per_gas: max_fee,
        max_priority_fee_per_gas: max_priority,
        to,
        value,
        access_list: Default::default(),
        input,
    };
    let sig_hash = tx.signature_hash();
    let sig = sign_message(B256::from(sk), sig_hash).expect("sign");
    let signed = tx.into_signed(sig);
    ArbTransactionSigned::from_envelope(EthereumTxEnvelope::Eip1559(signed))
}

pub fn sign_2930(
    chain_id: u64,
    nonce: u64,
    gas_price: u128,
    gas_limit: u64,
    to: TxKind,
    value: U256,
    input: Bytes,
    access_list: alloy_eips::eip2930::AccessList,
    sk: [u8; 32],
) -> ArbTransactionSigned {
    let tx = TxEip2930 {
        chain_id,
        nonce,
        gas_price,
        gas_limit,
        to,
        value,
        access_list,
        input,
    };
    let sig_hash = tx.signature_hash();
    let sig = sign_message(B256::from(sk), sig_hash).expect("sign");
    let signed = tx.into_signed(sig);
    ArbTransactionSigned::from_envelope(EthereumTxEnvelope::Eip2930(signed))
}

pub fn recover(tx: ArbTransactionSigned) -> Recovered<ArbTransactionSigned> {
    let sender = tx.recover_signer().expect("recover");
    Recovered::new_unchecked(tx, sender)
}

pub struct ExecutorScaffold {
    pub harness: ArbosHarness,
    pub chain_id: u64,
    pub base_fee: u64,
    pub block_number: u64,
    pub timestamp: u64,
}

impl ExecutorScaffold {
    pub fn new() -> Self {
        Self {
            harness: ArbosHarness::new()
                .with_arbos_version(30)
                .with_chain_id(CHAIN_ID)
                .initialize(),
            chain_id: CHAIN_ID,
            base_fee: 100_000_000,
            block_number: 1,
            timestamp: 1_700_000_000,
        }
    }

    pub fn with_funded(mut self, accounts: &[(Address, U256)]) -> Self {
        for (addr, bal) in accounts {
            fund_account(self.harness.state(), *addr, *bal);
        }
        self
    }

    pub fn evm_env(&self) -> EvmEnv<SpecId> {
        let mut env = EvmEnv {
            cfg_env: CfgEnv::default(),
            block_env: BlockEnv::default(),
        };
        env.cfg_env.chain_id = self.chain_id;
        env.cfg_env.disable_base_fee = true;
        env.block_env.timestamp = U256::from(self.timestamp);
        env.block_env.basefee = self.base_fee;
        env.block_env.gas_limit = 30_000_000;
        env.block_env.number = U256::from(self.block_number);
        env
    }

    pub fn evm_config(&self) -> ArbEvmConfig {
        let chain_spec: Arc<ChainSpec> = Arc::new(ChainSpec::default());
        ArbEvmConfig::new(chain_spec)
    }
}

impl Default for ExecutorScaffold {
    fn default() -> Self {
        Self::new()
    }
}

pub const RECIPIENT: Address = address!("11111111111111111111111111111111111111ff");
