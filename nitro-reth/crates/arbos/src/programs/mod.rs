use revm::Database;

use arb_storage::Storage;

/// Stylus programs state.
pub struct Programs<D> {
    pub backing_storage: Storage<D>,
    pub arbos_version: u64,
}

impl<D: Database> Programs<D> {
    pub fn initialize(_sto: &Storage<D>) {
        // TODO: implement full initialization
    }

    pub fn open(arbos_version: u64, sto: Storage<D>) -> Self {
        Self {
            backing_storage: sto,
            arbos_version,
        }
    }
}
