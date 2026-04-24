use arb_bench::{
    capture::synthetic::generate,
    runner::{in_process::InProcessRunner, RunnerConfig},
};
use criterion::{criterion_group, criterion_main, BenchmarkId, Criterion};

/// Wires up to the runner via the `stylus_deep_call_stack` synthetic generator.
/// The generator currently delegates to `transfer_train` until end-to-end
/// Stylus deploy is wired via the corpus; the bench is structured so the
/// upgrade is drop-in.
fn bench_stylus_dispatch(c: &mut Criterion) {
    let mut group = c.benchmark_group("stylus_dispatch");
    for txs_per_block in [4usize, 16] {
        let g = generate(
            "bench/stylus_dispatch",
            421614,
            60,
            "stylus_deep_call_stack",
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

criterion_group!(benches, bench_stylus_dispatch);
criterion_main!(benches);
