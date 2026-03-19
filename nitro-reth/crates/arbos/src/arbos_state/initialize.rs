use alloy_primitives::{Address, B256, U256};
use revm::Database;

use crate::{
    burn::Burner,
    retryables::{self, RetryableState},
};

use super::ArbosState;

/// Genesis data for a retryable ticket.
#[derive(Debug, Clone)]
pub struct InitRetryableData {
    pub id: B256,
    pub timeout: u64,
    pub from: Address,
    pub to: Option<Address>,
    pub callvalue: U256,
    pub beneficiary: Address,
    pub calldata: Vec<u8>,
}

/// Genesis data for an account.
#[derive(Debug, Clone)]
pub struct AccountInitInfo {
    pub addr: Address,
    pub nonce: u64,
    pub balance: U256,
    pub contract_info: Option<ContractInitInfo>,
    pub aggregator_info: Option<AggregatorInitInfo>,
}

/// Contract info for genesis account initialization.
#[derive(Debug, Clone)]
pub struct ContractInitInfo {
    pub code: Vec<u8>,
    pub storage: Vec<(U256, U256)>,
}

/// Aggregator (batch poster) info for genesis account initialization.
#[derive(Debug, Clone)]
pub struct AggregatorInitInfo {
    pub fee_collector: Address,
}

/// Creates a genesis block header.
///
/// Returns the fields needed for the genesis block. The actual block
/// construction uses reth's block types, so this returns a struct
/// that the genesis pipeline can consume.
#[derive(Debug, Clone)]
pub struct GenesisBlockInfo {
    pub parent_hash: B256,
    pub block_number: u64,
    pub timestamp: u64,
    pub state_root: B256,
    pub gas_limit: u64,
    pub base_fee: u64,
    pub nonce: u64,
    pub arbos_format_version: u64,
}

/// Build genesis block info from chain parameters.
pub fn make_genesis_block(
    parent_hash: B256,
    block_number: u64,
    timestamp: u64,
    state_root: B256,
    initial_arbos_version: u64,
) -> GenesisBlockInfo {
    use crate::l2_pricing;

    GenesisBlockInfo {
        parent_hash,
        block_number,
        timestamp,
        state_root,
        gas_limit: l2_pricing::GETH_BLOCK_GAS_LIMIT,
        base_fee: l2_pricing::INITIAL_BASE_FEE_WEI,
        nonce: 1, // genesis reads the init message
        arbos_format_version: initial_arbos_version,
    }
}

/// Initialize retryable tickets from genesis data.
///
/// Expired retryables (timeout <= current_timestamp) are skipped, and their
/// call value is returned as `(beneficiary, callvalue)` pairs for the caller
/// to credit balances. Active retryables are sorted by timeout and created.
///
/// Returns `(balance_credits, escrow_credits)` where:
/// - `balance_credits`: expired retryable beneficiaries to credit
/// - `escrow_credits`: (escrow_address, callvalue) for active retryable escrow funding
pub fn initialize_retryables<D: Database>(
    rs: &RetryableState<D>,
    mut retryables_data: Vec<InitRetryableData>,
    current_timestamp: u64,
) -> Result<(Vec<(Address, U256)>, Vec<(Address, U256)>), ()> {
    let mut balance_credits = Vec::new();
    let mut active_retryables = Vec::new();

    // Separate expired from active retryables.
    for r in retryables_data.drain(..) {
        if r.timeout <= current_timestamp {
            balance_credits.push((r.beneficiary, r.callvalue));
            continue;
        }
        active_retryables.push(r);
    }

    // Sort by timeout, then by id for determinism.
    active_retryables.sort_by(|a, b| a.timeout.cmp(&b.timeout).then_with(|| a.id.cmp(&b.id)));

    let mut escrow_credits = Vec::new();

    for r in &active_retryables {
        let escrow_addr = retryables::retryable_escrow_address(r.id);
        escrow_credits.push((escrow_addr, r.callvalue));
        rs.create_retryable(
            r.id,
            r.timeout,
            r.from,
            r.to,
            r.callvalue,
            r.beneficiary,
            &r.calldata,
        )?;
    }

    Ok((balance_credits, escrow_credits))
}

/// Initialize an account's ArbOS-specific state during genesis.
///
/// If the account has aggregator info and is a known batch poster,
/// sets the batch poster's pay-to (fee collector) address.
pub fn initialize_arbos_account<D: Database, B: Burner>(
    arbos_state: &ArbosState<D, B>,
    account: &AccountInitInfo,
) -> Result<(), ()> {
    if let Some(ref aggregator) = account.aggregator_info {
        let poster_table = arbos_state.l1_pricing_state.batch_poster_table();
        let is_poster = poster_table.contains_poster(account.addr)?;
        if is_poster {
            let poster = poster_table.open_poster(account.addr, false)?;
            poster.set_pay_to(aggregator.fee_collector)?;
        }
    }
    Ok(())
}

/// Full database initialization for ArbOS genesis.
///
/// This is the high-level orchestrator that:
/// 1. Initializes ArbOS state (version upgrades, precompile code)
/// 2. Adds chain owner
/// 3. Imports address table entries
/// 4. Imports retryable tickets
/// 5. Imports account state (balances, nonces, code, storage, batch poster config)
///
/// The caller provides the state database, init data, and handles commits.
/// Balance credits from expired retryables and escrow funding are returned
/// for the caller to execute against the state.
#[derive(Debug)]
pub struct GenesisInitResult {
    /// Expired retryable beneficiaries to credit.
    pub balance_credits: Vec<(Address, U256)>,
    /// Escrow addresses to fund for active retryables.
    pub escrow_credits: Vec<(Address, U256)>,
    /// Accounts to initialize (balances, nonces, code, storage).
    pub accounts: Vec<AccountInitInfo>,
}

/// Initialize ArbOS in the database.
///
/// Creates the ArbOS state, adds the chain owner, imports address table
/// entries, retryable tickets, and accounts. Returns a `GenesisInitResult`
/// containing all balance operations the caller needs to execute.
pub fn initialize_arbos_in_database<D: Database, B: Burner>(
    arbos_state: &ArbosState<D, B>,
    chain_owner: Address,
    address_table_entries: Vec<Address>,
    retryable_data: Vec<InitRetryableData>,
    accounts: Vec<AccountInitInfo>,
    current_timestamp: u64,
) -> Result<GenesisInitResult, ()> {
    // Add chain owner.
    if chain_owner != Address::ZERO {
        arbos_state.chain_owners.add(chain_owner)?;
    }

    // Import address table entries.
    let table_size = arbos_state.address_table.size()?;
    if table_size != 0 {
        return Err(());
    }
    for (i, addr) in address_table_entries.iter().enumerate() {
        let slot = arbos_state.address_table.register(*addr)?;
        if slot != i as u64 {
            return Err(());
        }
    }

    // Import retryable tickets.
    let (balance_credits, escrow_credits) = initialize_retryables(
        &arbos_state.retryable_state,
        retryable_data,
        current_timestamp,
    )?;

    // Initialize per-account ArbOS state (batch poster config).
    for account in &accounts {
        initialize_arbos_account(arbos_state, account)?;
    }

    Ok(GenesisInitResult {
        balance_credits,
        escrow_credits,
        accounts,
    })
}
