use arb_bench::{
    capture::synthetic::generate,
    runner::{RunnerConfig, in_process::InProcessRunner},
};
use criterion::{BenchmarkId, Criterion, Throughput, criterion_group, criterion_main};

fn bench_block_executor(c: &mut Criterion) {
    let mut group = c.benchmark_group("block_executor");
    for txs_per_block in [4usize, 64, 256] {
        let g = generate(
            "bench/block_executor",
            421614,
            30,
            "transfer_train",
            &serde_json::json!({ "block_count": 4, "txs_per_block": txs_per_block }),
        )
        .expect("generate");
        let total_txs = (g.blocks.len() * txs_per_block) as u64;
        group.throughput(Throughput::Elements(total_txs));
        group.bench_function(BenchmarkId::from_parameter(txs_per_block), |b| {
            b.iter(|| {
                let mut runner = InProcessRunner::new(RunnerConfig::default());
                let r = runner.run(g.clone()).expect("run");
                criterion::black_box(r.summary.total_gas);
            });
        });
    }
    group.finish();
}

criterion_group!(benches, bench_block_executor);
criterion_main!(benches);
