use arb_bench::{
    capture::synthetic::generate,
    runner::{in_process::InProcessRunner, RunnerConfig},
};
use criterion::{criterion_group, criterion_main, BenchmarkId, Criterion, Throughput};

/// Indirect proxy: runs `transfer_train` blocks with varying tx counts and
/// measures total wall-clock per run. State-root recomputation is the dominant
/// cost in the executor's `finish()` step, so this captures regressions in
/// trie hashing along with execution.
fn bench_state_root(c: &mut Criterion) {
    let mut group = c.benchmark_group("state_root");
    for txs_per_block in [8usize, 64, 256] {
        let g = generate(
            "bench/state_root",
            421614,
            30,
            "transfer_train",
            &serde_json::json!({ "block_count": 4, "txs_per_block": txs_per_block }),
        )
        .expect("generate");
        let total = (g.blocks.len() * txs_per_block) as u64;
        group.throughput(Throughput::Elements(total));
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

criterion_group!(benches, bench_state_root);
criterion_main!(benches);
