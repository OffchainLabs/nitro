use arbitrary::{Arbitrary, Unstructured};
use serde::Serialize;

/// ArbOS version selector drawn from the live set.
#[derive(Debug, Clone, Copy, Serialize)]
pub struct ArbosVersion(pub u64);

impl<'a> Arbitrary<'a> for ArbosVersion {
    fn arbitrary(u: &mut Unstructured<'a>) -> arbitrary::Result<Self> {
        const CANDIDATES: [u64; 11] = [10, 11, 20, 30, 31, 32, 40, 41, 50, 51, 60];
        let max_idx = CANDIDATES.len() - 1;
        let idx = u.int_in_range(0..=max_idx)?;
        Ok(ArbosVersion(CANDIDATES[idx]))
    }
}
