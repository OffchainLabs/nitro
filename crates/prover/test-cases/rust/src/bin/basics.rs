// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
fn main() {
	let mut x: i32 = 100;
	if std::env::vars().count() == 0 {
		x = x.wrapping_add(1);
	}
	std::process::exit(x ^ 101)
}
