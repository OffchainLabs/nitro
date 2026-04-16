use alloy_consensus::{Eip658Value, Receipt as AlloyReceipt, Typed2718};
use alloy_eips::eip2718::{Decodable2718, Encodable2718};
use alloy_primitives::{Address, Bytes, Log, LogData, B256};
use arb_primitives::{
    multigas::MultiGas,
    receipt::{ArbDepositReceipt, ArbReceipt, ArbReceiptKind},
};
use proptest::prelude::*;

fn arb_log() -> impl Strategy<Value = Log> {
    (
        prop::array::uniform20(any::<u8>()),
        prop::collection::vec(any::<[u8; 32]>(), 0..=4),
        prop::collection::vec(any::<u8>(), 0..64),
    )
        .prop_map(|(addr, topics, data)| Log {
            address: Address::from(addr),
            data: LogData::new(topics.into_iter().map(B256::from).collect(), Bytes::from(data))
                .expect("topics <= 4"),
        })
}

fn arb_receipt_inner() -> impl Strategy<Value = AlloyReceipt> {
    (any::<bool>(), any::<u64>(), prop::collection::vec(arb_log(), 0..3))
        .prop_map(|(status, cumulative_gas_used, logs)| AlloyReceipt {
            status: Eip658Value::Eip658(status),
            cumulative_gas_used,
            logs,
        })
}

fn arb_kind() -> impl Strategy<Value = ArbReceiptKind> {
    prop_oneof![
        arb_receipt_inner().prop_map(ArbReceiptKind::Legacy),
        arb_receipt_inner().prop_map(ArbReceiptKind::Eip1559),
        arb_receipt_inner().prop_map(ArbReceiptKind::Eip2930),
        arb_receipt_inner().prop_map(ArbReceiptKind::Eip7702),
        Just(ArbReceiptKind::Deposit(ArbDepositReceipt)),
        arb_receipt_inner().prop_map(ArbReceiptKind::Unsigned),
        arb_receipt_inner().prop_map(ArbReceiptKind::Contract),
        arb_receipt_inner().prop_map(ArbReceiptKind::Retry),
        arb_receipt_inner().prop_map(ArbReceiptKind::SubmitRetryable),
        arb_receipt_inner().prop_map(ArbReceiptKind::Internal),
    ]
}

proptest! {
    #[test]
    fn arb_receipt_2718_round_trip(kind in arb_kind()) {
        let r = ArbReceipt::new(kind);
        let mut buf = Vec::new();
        r.encode_2718(&mut buf);
        let mut slice = buf.as_slice();
        let decoded = ArbReceipt::decode_2718(&mut slice).expect("decode");
        prop_assert!(slice.is_empty());
        prop_assert_eq!(decoded.ty(), r.ty());
        prop_assert_eq!(decoded.kind, r.kind);
    }

    #[test]
    fn arb_receipt_decode_2718_fuzz_no_panic(bytes in prop::collection::vec(any::<u8>(), 0..256)) {
        let mut slice = bytes.as_slice();
        let _ = ArbReceipt::decode_2718(&mut slice);
    }
}

#[test]
fn arb_receipt_round_trip_with_metadata() {
    let mut r = ArbReceipt::new(ArbReceiptKind::Legacy(AlloyReceipt {
        status: Eip658Value::Eip658(true),
        cumulative_gas_used: 21_000,
        logs: vec![],
    }));
    r.gas_used_for_l1 = 12_345;
    r.multi_gas_used = MultiGas::zero();

    let mut buf = Vec::new();
    r.encode_2718(&mut buf);
    let mut slice = buf.as_slice();
    let decoded = ArbReceipt::decode_2718(&mut slice).expect("decode");
    assert_eq!(decoded.kind, r.kind);
}
