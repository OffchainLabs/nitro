use sha3::Keccak256;
use digest::Digest;

fn main() {
	let mut hasher = Keccak256::new();
	for i in 0..5 {
		hasher.update(&[i]);
	}
	let output: [u8; 32] = hasher.finalize().into();
	std::process::exit(output[0].into());
}
