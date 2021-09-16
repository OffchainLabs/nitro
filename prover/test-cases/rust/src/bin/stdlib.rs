use std::process::exit;

fn main() {
	let mut x = Vec::new();
	for i in 0..5 {
		x.push(i);
	}
	exit(x.into_iter().sum());
}
