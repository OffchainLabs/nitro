use alloy_primitives::{Address, U256};
use arbitrary::{Arbitrary, Unstructured};
use serde::Serialize;

#[derive(Debug, Clone, Arbitrary, Serialize)]
pub struct TxScenario {
    pub from: Address,
    pub to: Option<Address>,
    pub data: BoundedBytes<2048>,
    pub value: U256,
    pub gas: u64,
    pub max_fee: u128,
}

/// Byte vector capped at `N` bytes to bound fuzzer input size.
#[derive(Debug, Clone, Default, Serialize)]
pub struct BoundedBytes<const N: usize>(pub Vec<u8>);

impl<'a, const N: usize> Arbitrary<'a> for BoundedBytes<N> {
    fn arbitrary(u: &mut Unstructured<'a>) -> arbitrary::Result<Self> {
        let max = u.arbitrary_len::<u8>()?.min(N);
        let mut out = vec![0u8; max];
        u.fill_buffer(&mut out)?;
        Ok(Self(out))
    }
}

impl<const N: usize> AsRef<[u8]> for BoundedBytes<N> {
    fn as_ref(&self) -> &[u8] {
        &self.0
    }
}

impl<const N: usize> From<BoundedBytes<N>> for Vec<u8> {
    fn from(value: BoundedBytes<N>) -> Self {
        value.0
    }
}
