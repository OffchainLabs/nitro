use arb_bench::{
    capture::synthetic::generate,
    runner::{in_process::InProcessRunner, RunnerConfig},
};
use criterion::{criterion_group, criterion_main, BenchmarkId, Criterion};

/// Cold-cache Stylus dispatch. Same upgrade path as `stylus_dispatch`.
fn bench_stylus_ink_metering(c: &mut Criterion) {
    let mut group = c.benchmark_group("stylus_ink_metering");
    for txs_per_block in [4usize, 16] {
        let g = generate(
            "bench/stylus_ink_metering",
            421614,
            60,
            "stylus_cold_cache",
            &serde_json::json!({ "block_count": 2, "txs_per_block": txs_per_block }),
        )
        .expect("generate");
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

criterion_group!(benches, bench_stylus_ink_metering);
criterion_main!(benches);
