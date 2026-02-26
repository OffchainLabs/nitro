mod arbgasinfo;
mod arbinfo;
mod arbsys;
mod storage_slot;

pub use arbgasinfo::{create_arbgasinfo_precompile, ARBGASINFO_ADDRESS};
pub use arbinfo::{create_arbinfo_precompile, ARBINFO_ADDRESS};
pub use arbsys::{
    create_arbsys_precompile, get_cached_l1_block_number, set_cached_l1_block_number,
    store_arbsys_state, take_arbsys_state, ArbSysMerkleState, ARBSYS_ADDRESS,
};
pub use storage_slot::{compute_storage_slot, ARBOS_STATE_ADDRESS};

use alloy_evm::precompiles::PrecompilesMap;

/// Register all Arbitrum precompiles into a [`PrecompilesMap`].
pub fn register_arb_precompiles(map: &mut PrecompilesMap) {
    map.extend_precompiles([
        (ARBSYS_ADDRESS, create_arbsys_precompile()),
        (ARBGASINFO_ADDRESS, create_arbgasinfo_precompile()),
        (ARBINFO_ADDRESS, create_arbinfo_precompile()),
    ]);
}
