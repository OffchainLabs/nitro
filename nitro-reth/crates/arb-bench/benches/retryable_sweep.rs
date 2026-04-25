use arb_bench::{
    capture::synthetic::generate,
    runner::{in_process::InProcessRunner, RunnerConfig},
};
use criterion::{criterion_group, criterion_main, BenchmarkId, Criterion};

fn bench_retryable_sweep(c: &mut Criterion) {
    let mut group = c.benchmark_group("retryable_sweep");
    for blocks in [4usize, 16, 64] {
        let g = generate(
            "bench/retryable_sweep",
            421614,
            30,
            "retryable_timeout_sweep",
            &serde_json::json!({ "block_count": blocks, "txs_per_block": 8 }),
        )
        .expect("generate");
        group.bench_function(BenchmarkId::from_parameter(blocks), |b| {
            b.iter(|| {
                let mut runner = InProcessRunner::new(RunnerConfig::default());
                let r = runner.run(g.clone()).expect("run");
                criterion::black_box(r.summary.total_gas);
            });
        });
    }
    group.finish();
}

criterion_group!(benches, bench_retryable_sweep);
criterion_main!(benches);
