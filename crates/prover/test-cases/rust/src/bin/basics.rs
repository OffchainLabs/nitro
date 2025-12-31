fn main() {
	let mut x: i32 = 100;
	if std::env::vars().count() == 0 {
		x = x.wrapping_add(1);
	}
	std::process::exit(x ^ 101)
}
