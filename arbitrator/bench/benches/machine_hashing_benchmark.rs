use criterion::{criterion_group, criterion_main, Criterion};

use bench::prepare::*;

pub fn machine_hashing_benchmark(c: &mut Criterion) {
    let mut group = c.benchmark_group("group");
    group.measurement_time(std::time::Duration::from_secs(60));
    let mut machine = prepare_machine().unwrap();
    let step_size = 16_384;
    group.bench_function("machine hashing step size 16384", |b| {
        b.iter(|| {
            machine.step_n(step_size).unwrap();
            let _ = machine.hash();
        })
    });
    group.finish();
}

criterion_group!(benches, machine_hashing_benchmark);
criterion_main!(benches);
