#[derive(Debug, Clone, Copy, PartialEq, Eq)]
#[derive(Default)]
pub enum FixtureMode {
    #[default]
    Verify,
    Record,
    Compare,
}

impl FixtureMode {
    pub fn from_env() -> Self {
        match std::env::var("ARB_SPEC_MODE").as_deref() {
            Ok("record") => FixtureMode::Record,
            Ok("compare") => FixtureMode::Compare,
            _ => FixtureMode::Verify,
        }
    }
}

