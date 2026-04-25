use alloy_eips::eip2718::{Decodable2718, Encodable2718};
use arb_bench::capture::synthetic::generate;
use arb_primitives::ArbTransactionSigned;
use criterion::{criterion_group, criterion_main, BenchmarkId, Criterion, Throughput};

fn corpus(txs_per_block: usize) -> Vec<Vec<u8>> {
    let g = generate(
        "bench/tx_decode",
        421614,
        30,
        "transfer_train",
        &serde_json::json!({ "block_count": 4, "txs_per_block": txs_per_block }),
    )
    .expect("generate");
    let mut bytes = Vec::new();
    for b in g.blocks {
        for tx in b.txs {
            let mut buf = Vec::new();
            tx.encode_2718(&mut buf);
            bytes.push(buf);
        }
    }
    bytes
}

fn bench_tx_decode(c: &mut Criterion) {
    let mut group = c.benchmark_group("tx_decode");
    for n in [1usize, 16, 256] {
        let raws = corpus(n);
        group.throughput(Throughput::Elements(raws.len() as u64));
        group.bench_function(BenchmarkId::from_parameter(n), |b| {
            b.iter(|| {
                for raw in &raws {
                    let tx =
                        ArbTransactionSigned::decode_2718(&mut raw.as_slice()).expect("decode");
                    criterion::black_box(tx);
                }
            });
        });
    }
    group.finish();
}

criterion_group!(benches, bench_tx_decode);
criterion_main!(benches);
