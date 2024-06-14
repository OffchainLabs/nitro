use crate::{full_payload::NsIndex, namespace_payload::types::TxIndex};
use serde::{Deserialize, Serialize};

#[derive(Clone, Debug, Eq, PartialEq, Serialize, Deserialize)]
pub struct Index {
    ns_index: NsIndex,
    tx_index: TxIndex,
}

// TODO don't impl `PartialOrd`
// It's needed only for `QueryablePayload` trait:
// https://github.com/EspressoSystems/hotshot-query-service/issues/639
impl PartialOrd for Index {
    fn partial_cmp(&self, _other: &Self) -> Option<std::cmp::Ordering> {
        Some(self.cmp(_other))
    }
}
// TODO don't impl `Ord`
// It's needed only for `QueryablePayload` trait:
// https://github.com/EspressoSystems/hotshot-query-service/issues/639
impl Ord for Index {
    fn cmp(&self, _other: &Self) -> std::cmp::Ordering {
        unimplemented!()
    }
}
