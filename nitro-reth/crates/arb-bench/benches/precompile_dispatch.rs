use arb_bench::{
    capture::synthetic::generate,
    runner::{in_process::InProcessRunner, RunnerConfig},
};
use criterion::{criterion_group, criterion_main, BenchmarkId, Criterion, Throughput};

fn bench_precompile_dispatch(c: &mut Criterion) {
    let mut group = c.benchmark_group("precompile_dispatch");
    for txs_per_block in [16usize, 64, 256] {
        let g = generate(
            "bench/precompile_dispatch",
            421614,
            30,
            "precompile_fanout",
            &serde_json::json!({ "block_count": 2, "txs_per_block": txs_per_block }),
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

criterion_group!(benches, bench_precompile_dispatch);
criterion_main!(benches);
