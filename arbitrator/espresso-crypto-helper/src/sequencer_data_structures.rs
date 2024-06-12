// TODO import from sequencer: https://github.com/EspressoSystems/nitro-espresso-integration/issues/87
// This module is essentially copy and pasted VID logic from the sequencer repo. It is an unfortunate workaround
// until the VID portion of the sequencer repo is WASM-compatible.
use ark_ff::{BigInteger, PrimeField};
use ark_serialize::{
    CanonicalDeserialize, CanonicalSerialize, Compress, Read, SerializationError, Valid, Validate,
};
use bytesize::ByteSize;
use committable::{Commitment, Committable, RawCommitmentBuilder};
use derive_more::{Add, Display, From, Into, Sub};
use digest::OutputSizeUser;
use either::Either;
use ethers_core::{
    types::{Address, Signature, H256, U256},
    utils::{parse_units, ParseUnits},
};
use jf_merkle_tree::{
    prelude::{LightWeightSHA3MerkleTree, Sha3Digest, Sha3Node},
    universal_merkle_tree::UniversalMerkleTree,
    MerkleTreeScheme, ToTraversalPath,
};
use num_traits::PrimInt;
use serde::{Deserialize, Serialize};
use std::{default::Default, str::FromStr};
use tagged_base64::tagged;
use trait_set::trait_set;
use typenum::Unsigned;

use crate::{full_payload::NsTable, hotshot_types::VidCommitment};
use crate::{
    utils::{impl_serde_from_string_or_integer, Err, FromStringOrInteger},
    NamespaceId,
};

trait_set! {
    pub trait TableWordTraits = CanonicalSerialize
        + CanonicalDeserialize
        + TryFrom<usize>
        + TryInto<usize>
        + Default
         + PrimInt
        + std::marker::Sync;

    // Note: this trait is not used yet as for now the Payload structs are only parametrized with the TableWord parameter.
    pub trait OffsetTraits = CanonicalSerialize
        + CanonicalDeserialize
        + TryFrom<usize>
        + TryInto<usize>
        + Default
        + std::marker::Sync;

    // Note: this trait is not used yet as for now the Payload structs are only parametrized with the TableWord parameter.
    pub trait NsIdTraits =CanonicalSerialize + CanonicalDeserialize + Default + std::marker::Sync;
}

pub struct Transaction {
    namespace: NamespaceId,
    pub payload: Vec<u8>,
}

impl Transaction {
    pub fn new(namespace: NamespaceId, payload: Vec<u8>) -> Self {
        Self { namespace, payload }
    }

    pub fn namespace(&self) -> NamespaceId {
        self.namespace
    }

    pub fn payload(&self) -> &[u8] {
        &self.payload
    }

    pub fn into_payload(self) -> Vec<u8> {
        self.payload
    }
}

#[derive(Default, Hash, Copy, Clone, Debug, PartialEq, Eq, From, Into, Display)]
#[display(fmt = "{_0}")]

pub struct ChainId(U256);

impl From<u64> for ChainId {
    fn from(id: u64) -> Self {
        Self(id.into())
    }
}

impl FromStringOrInteger for ChainId {
    type Binary = U256;
    type Integer = u64;

    fn from_binary(b: Self::Binary) -> Result<Self, Err> {
        Ok(Self(b))
    }

    fn from_integer(i: Self::Integer) -> Result<Self, Err> {
        Ok(i.into())
    }

    fn from_string(s: String) -> Result<Self, Err> {
        if s.starts_with("0x") {
            Ok(Self(U256::from_str(&s).unwrap()))
        } else {
            Ok(Self(U256::from_dec_str(&s).unwrap()))
        }
    }

    fn to_binary(&self) -> Result<Self::Binary, Err> {
        Ok(self.0)
    }

    fn to_string(&self) -> Result<String, Err> {
        Ok(format!("{self}"))
    }
}

macro_rules! impl_to_fixed_bytes {
    ($struct_name:ident, $type:ty) => {
        impl $struct_name {
            pub(crate) fn to_fixed_bytes(self) -> [u8; core::mem::size_of::<$type>()] {
                let mut bytes = [0u8; core::mem::size_of::<$type>()];
                self.0.to_little_endian(&mut bytes);
                bytes
            }
        }
    };
}

impl_serde_from_string_or_integer!(ChainId);
impl_to_fixed_bytes!(ChainId, U256);

impl From<u16> for ChainId {
    fn from(id: u16) -> Self {
        Self(id.into())
    }
}

#[derive(Hash, Copy, Clone, Debug, Default, Display, PartialEq, Eq, From, Into)]
#[display(fmt = "{_0}")]
pub struct BlockSize(u64);

impl_serde_from_string_or_integer!(BlockSize);

impl FromStringOrInteger for BlockSize {
    type Binary = u64;
    type Integer = u64;

    fn from_binary(b: Self::Binary) -> Result<Self, Err> {
        Ok(Self(b))
    }

    fn from_integer(i: Self::Integer) -> Result<Self, Err> {
        Ok(Self(i))
    }

    fn from_string(s: String) -> Result<Self, Err> {
        Ok(BlockSize(s.parse::<ByteSize>().unwrap().0))
    }

    fn to_binary(&self) -> Result<Self::Binary, Err> {
        Ok(self.0)
    }

    fn to_string(&self) -> Result<String, Err> {
        Ok(format!("{self}"))
    }
}

/// Global variables for an Espresso blockchain.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize)]
pub struct ChainConfig {
    /// Espresso chain ID
    chain_id: ChainId,
    /// Maximum size in bytes of a block
    max_block_size: BlockSize,
    /// Minimum fee in WEI per byte of payload
    base_fee: FeeAmount,
    fee_contract: Option<Address>,
    fee_recipient: FeeAccount,
}

impl Committable for ChainConfig {
    fn tag() -> String {
        "CHAIN_CONFIG".to_string()
    }

    fn commit(&self) -> Commitment<Self> {
        let comm = committable::RawCommitmentBuilder::new(&Self::tag())
            .fixed_size_field("chain_id", &self.chain_id.to_fixed_bytes())
            .u64_field("max_block_size", self.max_block_size.0)
            .fixed_size_field("base_fee", &self.base_fee.to_fixed_bytes())
            .fixed_size_field("fee_recipient", &self.fee_recipient.to_fixed_bytes());
        let comm = if let Some(addr) = self.fee_contract {
            comm.u64_field("fee_contract", 1).fixed_size_bytes(&addr.0)
        } else {
            comm.u64_field("fee_contract", 0)
        };
        comm.finalize()
    }
}

#[derive(Clone, Debug, Copy, PartialEq, Deserialize, Serialize, Eq, Hash)]
pub struct ResolvableChainConfig {
    chain_config: Either<ChainConfig, Commitment<ChainConfig>>,
}

impl ResolvableChainConfig {
    pub fn commit(&self) -> Commitment<ChainConfig> {
        match self.chain_config {
            Either::Left(config) => config.commit(),
            Either::Right(commitment) => commitment,
        }
    }
    pub fn resolve(self) -> Option<ChainConfig> {
        match self.chain_config {
            Either::Left(config) => Some(config),
            Either::Right(_) => None,
        }
    }
}

impl From<Commitment<ChainConfig>> for ResolvableChainConfig {
    fn from(value: Commitment<ChainConfig>) -> Self {
        Self {
            chain_config: Either::Right(value),
        }
    }
}

impl From<ChainConfig> for ResolvableChainConfig {
    fn from(value: ChainConfig) -> Self {
        Self {
            chain_config: Either::Left(value),
        }
    }
}

pub type BlockMerkleTree = LightWeightSHA3MerkleTree<Commitment<Header>>;
pub type BlockMerkleCommitment = <BlockMerkleTree as MerkleTreeScheme>::Commitment;

#[tagged("BUILDER_COMMITMENT")]
#[derive(Clone, Debug, Hash, PartialEq, Eq, CanonicalSerialize, CanonicalDeserialize)]
/// Commitment that builders use to sign block options.
/// A thin wrapper around a Sha256 digest.
pub struct BuilderCommitment(Sha256Digest);

impl AsRef<Sha256Digest> for BuilderCommitment {
    fn as_ref(&self) -> &Sha256Digest {
        &self.0
    }
}

// Header values
#[derive(Clone, Debug, Deserialize, Serialize, Hash, PartialEq, Eq)]
pub struct Header {
    pub chain_config: ResolvableChainConfig,
    pub height: u64,
    pub timestamp: u64,
    pub l1_head: u64,
    pub l1_finalized: Option<L1BlockInfo>,
    pub payload_commitment: VidCommitment,
    pub builder_commitment: BuilderCommitment,
    pub ns_table: NsTable,
    pub block_merkle_tree_root: BlockMerkleCommitment,
    pub fee_merkle_tree_root: FeeMerkleCommitment,
    pub builder_signature: Option<Signature>,
    pub fee_info: FeeInfo,
}

impl Committable for Header {
    fn commit(&self) -> Commitment<Self> {
        let mut bmt_bytes = vec![];
        self.block_merkle_tree_root
            .serialize_with_mode(&mut bmt_bytes, ark_serialize::Compress::Yes)
            .unwrap();
        let mut fmt_bytes = vec![];
        self.fee_merkle_tree_root
            .serialize_with_mode(&mut fmt_bytes, ark_serialize::Compress::Yes)
            .unwrap();

        RawCommitmentBuilder::new(&Self::tag())
            .field("chain_config", self.chain_config.commit())
            .u64_field("height", self.height)
            .u64_field("timestamp", self.timestamp)
            .u64_field("l1_head", self.l1_head)
            .optional("l1_finalized", &self.l1_finalized)
            .constant_str("payload_commitment")
            .fixed_size_bytes(self.payload_commitment.as_ref().as_ref())
            .constant_str("builder_commitment")
            .fixed_size_bytes(self.builder_commitment.as_ref())
            .field("ns_table", self.ns_table.commit())
            .var_size_field("block_merkle_tree_root", &bmt_bytes)
            .var_size_field("fee_merkle_tree_root", &fmt_bytes)
            .field("fee_info", self.fee_info.commit())
            .finalize()
    }

    fn tag() -> String {
        // We use the tag "BLOCK" since blocks are identified by the hash of their header. This will
        // thus be more intuitive to users than "HEADER".
        "BLOCK".into()
    }
}

pub type FeeMerkleTree = UniversalMerkleTree<FeeAmount, Sha3Digest, FeeAccount, 256, Sha3Node>;
pub type FeeMerkleCommitment = <FeeMerkleTree as MerkleTreeScheme>::Commitment;

/// Type alias for byte array of SHA256 digest length
type Sha256Digest = [u8; <sha2::Sha256 as OutputSizeUser>::OutputSize::USIZE];

#[derive(Default, Hash, Copy, Clone, Debug, PartialEq, Eq, Add, Sub, From, Into, Display)]
#[display(fmt = "{_0}")]
pub struct FeeAmount(U256);

impl FromStringOrInteger for FeeAmount {
    type Binary = U256;
    type Integer = u64;

    fn from_binary(b: Self::Binary) -> Result<Self, Err> {
        Ok(Self(b))
    }

    fn from_integer(i: Self::Integer) -> Result<Self, Err> {
        Ok(i.into())
    }

    fn from_string(s: String) -> Result<Self, Err> {
        // For backwards compatibility, we have an ad hoc parser for WEI amounts represented as hex
        // strings.
        if let Some(s) = s.strip_prefix("0x") {
            return Ok(Self(s.parse().unwrap()));
        }

        // Strip an optional non-numeric suffix, which will be interpreted as a unit.
        let (base, unit) = s
            .split_once(char::is_whitespace)
            .unwrap_or((s.as_str(), "wei"));
        match parse_units(base, unit).unwrap() {
            ParseUnits::U256(n) => Ok(Self(n)),
            ParseUnits::I256(_) => panic!("amount cannot be negative"),
        }
    }

    fn to_binary(&self) -> Result<Self::Binary, Err> {
        Ok(self.0)
    }

    fn to_string(&self) -> Result<String, Err> {
        Ok(format!("{self}"))
    }
}

impl_serde_from_string_or_integer!(FeeAmount);
impl_to_fixed_bytes!(FeeAmount, U256);

impl From<u64> for FeeAmount {
    fn from(amt: u64) -> Self {
        Self(amt.into())
    }
}

impl CanonicalSerialize for FeeAmount {
    fn serialize_with_mode<W: std::io::prelude::Write>(
        &self,
        mut writer: W,
        _compress: Compress,
    ) -> Result<(), SerializationError> {
        Ok(writer.write_all(&self.to_fixed_bytes())?)
    }

    fn serialized_size(&self, _compress: Compress) -> usize {
        core::mem::size_of::<U256>()
    }
}
impl CanonicalDeserialize for FeeAmount {
    fn deserialize_with_mode<R: Read>(
        mut reader: R,
        _compress: Compress,
        _validate: Validate,
    ) -> Result<Self, SerializationError> {
        let mut bytes = [0u8; core::mem::size_of::<U256>()];
        reader.read_exact(&mut bytes)?;
        let value = U256::from_little_endian(&bytes);
        Ok(Self(value))
    }
}

impl CanonicalSerialize for FeeAccount {
    fn serialize_with_mode<W: std::io::prelude::Write>(
        &self,
        mut writer: W,
        _compress: Compress,
    ) -> Result<(), SerializationError> {
        Ok(writer.write_all(&self.0.to_fixed_bytes())?)
    }

    fn serialized_size(&self, _compress: Compress) -> usize {
        core::mem::size_of::<Address>()
    }
}

impl CanonicalDeserialize for FeeAccount {
    fn deserialize_with_mode<R: Read>(
        mut reader: R,
        _compress: Compress,
        _validate: Validate,
    ) -> Result<Self, SerializationError> {
        let mut bytes = [0u8; core::mem::size_of::<Address>()];
        reader.read_exact(&mut bytes)?;
        let value = Address::from_slice(&bytes);
        Ok(Self(value))
    }
}

#[derive(
    Default,
    Hash,
    Copy,
    Clone,
    Debug,
    Display,
    Deserialize,
    Serialize,
    PartialEq,
    Eq,
    PartialOrd,
    Ord,
    From,
    Into,
)]
#[display(fmt = "{_0:x}")]
pub struct FeeAccount(Address);

impl FeeAccount {
    /// Return inner `Address`
    pub fn address(&self) -> Address {
        self.0
    }
    /// Return byte slice representation of inner `Address` type
    pub fn as_bytes(&self) -> &[u8] {
        self.0.as_bytes()
    }
    /// Return array containing underlying bytes of inner `Address` type
    pub fn to_fixed_bytes(self) -> [u8; 20] {
        self.0.to_fixed_bytes()
    }
}

impl ToTraversalPath<256> for FeeAccount {
    fn to_traversal_path(&self, height: usize) -> Vec<usize> {
        self.0
            .to_fixed_bytes()
            .into_iter()
            .take(height)
            .map(|i| i as usize)
            .collect()
    }
}

impl Valid for FeeAmount {
    fn check(&self) -> Result<(), SerializationError> {
        Ok(())
    }
}

impl Valid for FeeAccount {
    fn check(&self) -> Result<(), SerializationError> {
        Ok(())
    }
}

#[derive(
    Hash,
    Copy,
    Clone,
    Debug,
    Deserialize,
    Serialize,
    PartialEq,
    Eq,
    CanonicalSerialize,
    CanonicalDeserialize,
)]
pub struct FeeInfo {
    account: FeeAccount,
    amount: FeeAmount,
}

#[derive(Clone, Copy, Debug, Default, Deserialize, Serialize, Hash, PartialEq, Eq)]
pub struct L1BlockInfo {
    pub number: u64,
    pub timestamp: U256,
    pub hash: H256,
}

impl Committable for L1BlockInfo {
    fn commit(&self) -> Commitment<Self> {
        let mut timestamp = [0u8; 32];
        self.timestamp.to_little_endian(&mut timestamp);

        RawCommitmentBuilder::new(&Self::tag())
            .u64_field("number", self.number)
            // `RawCommitmentBuilder` doesn't have a `u256_field` method, so we simulate it:
            .constant_str("timestamp")
            .fixed_size_bytes(&timestamp)
            .constant_str("hash")
            .fixed_size_bytes(&self.hash.0)
            .finalize()
    }

    fn tag() -> String {
        "L1BLOCK".into()
    }
}

impl Committable for FeeInfo {
    fn commit(&self) -> Commitment<Self> {
        RawCommitmentBuilder::new(&Self::tag())
            .fixed_size_field("account", &self.account.to_fixed_bytes())
            .fixed_size_field("amount", &self.amount.to_fixed_bytes())
            .finalize()
    }
    fn tag() -> String {
        "FEE_INFO".into()
    }
}

pub fn field_to_u256<F: PrimeField>(f: F) -> U256 {
    if F::MODULUS_BIT_SIZE > 256 {
        panic!("Shouldn't convert a >256-bit field to U256");
    }
    U256::from_little_endian(&f.into_bigint().to_bytes_le())
}

#[cfg(test)]
mod tests {
    use committable::Committable;

    use super::Header;

    #[test]
    fn header_test() {
        let header_str = include_str!("./mock_data/header.json");
        let header = serde_json::from_str::<Header>(&header_str).unwrap();
        // Copied from espresso sequencer reference test
        let expected = "BLOCK~6Ol30XYkdKaNFXw0QAkcif18Lk8V8qkC4M81qTlwL707";
        assert_eq!(header.commit().to_string(), expected);
    }
}
