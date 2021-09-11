#![feature(start)]
#![no_std]

extern crate test_cases;

use sha3::Keccak256;
use digest::Digest;

#[start]
fn main(argc: isize, _: *const *const u8) -> isize {
	let mut hasher = Keccak256::new();
	for i in 0..(argc as u8) {
		hasher.update(&[i]);
	}
	let output: [u8; 32] = hasher.finalize().into();
	output[0].into()
}
