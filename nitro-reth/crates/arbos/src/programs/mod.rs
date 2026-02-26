pub mod data_pricer;

use revm::Database;

use arb_storage::Storage;

use self::data_pricer::{init_data_pricer, open_data_pricer, DataPricer};

/// Stylus programs state.
pub struct Programs<D> {
    pub backing_storage: Storage<D>,
    pub arbos_version: u64,
    pub data_pricer: DataPricer<D>,
}

impl<D: Database> Programs<D> {
    pub fn initialize(sto: &Storage<D>) {
        let data_pricer_sto = sto.open_sub_storage(&[0]);
        init_data_pricer(&data_pricer_sto);
    }

    pub fn open(arbos_version: u64, sto: Storage<D>) -> Self {
        let data_pricer_sto = sto.open_sub_storage(&[0]);
        let data_pricer = open_data_pricer(&data_pricer_sto);
        Self {
            backing_storage: sto,
            arbos_version,
            data_pricer,
        }
    }
}
