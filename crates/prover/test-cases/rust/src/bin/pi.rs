fn main() {
    // Compute π with Ramanujan's approximation

    macro_rules! compute {
		($t:ty) => {
			let mut estimate = 0.;
			let multiplier = 2. * (2 as $t).sqrt() / 9801.;
			let mut fact = 1.;
			let mut quad_fact = 1.;
			let mut k = 0. as $t;
			while k <= 10. {
				let quad = k * 4.;
				if k > 0. {
					fact *= k;
					quad_fact *= quad - 3.;
					quad_fact *= quad - 2.;
					quad_fact *= quad - 1.;
					quad_fact *= quad;
				}
				let fact_square = fact * fact;
				estimate += multiplier * quad_fact * (1103. + 26390. * k) / (fact_square * fact_square * (396 as $t).powf(quad));
				let pi = 1./estimate;
				println!("Estimation with {} after iteration {}: {}", stringify!($t), k, pi);
				assert!(pi.is_nan() || (pi > 3.13 && pi < 3.15));
				k += 1.;
			}
		}
	}

    compute!(f32);
    compute!(f64);
}
