// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
fn main() {
	let mut x = Vec::new();
	for i in 1..10 {
		x.push(i);
	}
	let sum: usize = x.iter().cloned().sum();
	let product = x.into_iter().fold(1, |p, x| p * x);
	println!("Sum: {}", sum);
	eprintln!("Product: {}", product);
	assert_eq!(sum, 45);
	assert_eq!(product, 362880);
}
