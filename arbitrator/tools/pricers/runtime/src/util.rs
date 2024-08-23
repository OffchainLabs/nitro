// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use rand::{distributions::Standard, prelude::Distribution, Rng};

pub fn random_vec<T>(len: usize) -> Vec<T>
where
    Standard: Distribution<T>,
{
    let mut rng = rand::thread_rng();
    let mut entropy = Vec::with_capacity(len);
    for _ in 0..len {
        entropy.push(rng.gen())
    }
    entropy
}
