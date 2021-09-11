#![feature(start)]
#![no_std]

extern crate test_cases;

#[start]
fn main(argc: isize, _: *const *const u8) -> isize {
	let mut x: isize = 100;
	if argc == 0 {
		x = x.wrapping_add(1);
	}
	x
}
