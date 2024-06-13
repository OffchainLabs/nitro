use crate::hotshot_types::SmallRangeProofType;
use crate::namespace_payload::types::{NumTxsUnchecked, TxIndex, TxTableEntries};
use serde::{Deserialize, Serialize};

/// Proof of correctness for transaction bytes in a block.
#[derive(Clone, Debug, Eq, PartialEq, Serialize, Deserialize)]
pub struct TxProof {
    // Naming conventions for this struct's fields:
    // - `payload_x`: bytes from the payload
    // - `payload_proof_x`: a proof of those bytes from the payload
    tx_index: TxIndex,

    // Number of txs declared in the tx table
    payload_num_txs: NumTxsUnchecked,
    payload_proof_num_txs: SmallRangeProofType,

    // Tx table entries for this tx
    payload_tx_table_entries: TxTableEntries,
    payload_proof_tx_table_entries: SmallRangeProofType,

    // This tx's payload bytes.
    // `None` if this tx has zero length.
    payload_proof_tx: Option<SmallRangeProofType>,
}
