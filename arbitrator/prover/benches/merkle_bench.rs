use arbutil::Bytes32;
use criterion::{criterion_group, criterion_main, Criterion};
use prover::merkle::{Merkle, MerkleType};
use rand::Rng;

fn extend_and_set_leavees(mut merkle: Merkle, rng: &mut rand::rngs::ThreadRng) {
    let new_leaves = vec![
        Bytes32::from([6; 32]),
        Bytes32::from([7; 32]),
        Bytes32::from([8; 32]),
        Bytes32::from([9; 32]),
        Bytes32::from([10; 32]),
    ];

    for _ in 0..100 {
        merkle.extend(new_leaves.clone()).expect("extend failed");
        for _ in 0..(merkle.len() / 10) {
            let random_index = rng.gen_range(0..merkle.len());
            merkle.set(random_index, Bytes32::from([rng.gen_range(0u8..9); 32]));
        }
    }
    merkle.root();
}

fn merkle_benchmark(c: &mut Criterion) {
    let mut rng = rand::thread_rng();
    let leaves = vec![
        Bytes32::from([1; 32]),
        Bytes32::from([2; 32]),
        Bytes32::from([3; 32]),
        Bytes32::from([4; 32]),
        Bytes32::from([5; 32]),
    ];

    // Perform many calls to set leaves to new values
    c.bench_function("extend_set_leaves_and_root", |b| {
        b.iter(|| {
            let merkle =
                Merkle::new_advanced(MerkleType::Memory, leaves.clone(), Bytes32::default(), 20);
            extend_and_set_leavees(merkle.clone(), &mut rng);
        })
    });
}

fn merkle_construction(c: &mut Criterion) {
    let mut rng = rand::thread_rng();
    let mut leaves: Vec<Bytes32> = Vec::with_capacity(1 >> 20);
    for _ in 0..(1 >> 20) {
        leaves.push(Bytes32::from([rng.gen_range(0u8..9); 32]));
    }

    c.bench_function("merkle_construction", |b| {
        b.iter(|| {
            let merkle =
                Merkle::new_advanced(MerkleType::Memory, leaves.clone(), Bytes32::default(), 21);
            merkle.root();
        })
    });
}

criterion_group!(benches, merkle_benchmark, merkle_construction);
criterion_main!(benches);
