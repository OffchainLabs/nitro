use arb_spec_tests::{run_dir, runner::fixtures_root};

macro_rules! spec_dir {
    ($name:ident, $dir:literal) => {
        #[test]
        fn $name() {
            run_dir(&fixtures_root().join($dir));
        }
    };
}

spec_dir!(pricing, "pricing");
spec_dir!(state_transitions, "state_transitions");
