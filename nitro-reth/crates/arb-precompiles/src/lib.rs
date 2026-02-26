mod arbfunctiontable;
mod arbgasinfo;
mod arbinfo;
mod arbosacts;
mod arbownerpublic;
mod arbstatistics;
mod arbsys;
mod storage_slot;

pub use arbfunctiontable::{create_arbfunctiontable_precompile, ARBFUNCTIONTABLE_ADDRESS};
pub use arbgasinfo::{create_arbgasinfo_precompile, ARBGASINFO_ADDRESS};
pub use arbinfo::{create_arbinfo_precompile, ARBINFO_ADDRESS};
pub use arbosacts::{create_arbosacts_precompile, ARBOSACTS_ADDRESS};
pub use arbownerpublic::{create_arbownerpublic_precompile, ARBOWNERPUBLIC_ADDRESS};
pub use arbstatistics::{create_arbstatistics_precompile, ARBSTATISTICS_ADDRESS};
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
        (ARBSTATISTICS_ADDRESS, create_arbstatistics_precompile()),
        (
            ARBFUNCTIONTABLE_ADDRESS,
            create_arbfunctiontable_precompile(),
        ),
        (ARBOSACTS_ADDRESS, create_arbosacts_precompile()),
        (
            ARBOWNERPUBLIC_ADDRESS,
            create_arbownerpublic_precompile(),
        ),
    ]);
}
