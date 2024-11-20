use crate::{full_payload::NsIndex, namespace_payload::types::TxIndex};
use serde::{Deserialize, Serialize};

#[derive(Clone, Debug, Eq, PartialEq, Serialize, Deserialize)]
pub struct Index {
    ns_index: NsIndex,
    tx_index: TxIndex,
}
