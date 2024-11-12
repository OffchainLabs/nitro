// TODO import from sequencer: https://github.com/EspressoSystems/nitro-espresso-integration/issues/87
// This module is essentially copy and pasted VID logic from the sequencer repo. It is an unfortunate workaround
// until the VID portion of the sequencer repo is WASM-compatible.
use ark_ff::{BigInteger, PrimeField};
use ark_serialize::{
    CanonicalDeserialize, CanonicalSerialize, Compress, Read, SerializationError, Valid, Validate,
};
use bytesize::ByteSize;
use committable::{Commitment, Committable, RawCommitmentBuilder};
use derive_more::Deref;
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
use serde::Serializer;
use serde::{
    de::{self, MapAccess, SeqAccess, Visitor},
    Deserialize, Deserializer, Serialize,
};
use serde_json::{Map, Value};
use std::{default::Default, fmt, str::FromStr};
use tagged_base64::tagged;
use trait_set::trait_set;
use typenum::Unsigned;

use crate::v0_3;
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
    pub namespace: NamespaceId,
    pub payload: Vec<u8>,
}

impl Transaction {
    pub fn new(namespace: NamespaceId, payload: Vec<u8>) -> Self {
        Self { namespace, payload }
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

#[derive(Hash, Copy, Clone, Debug, Default, Display, PartialEq, Eq, From, Into, Deref)]
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

pub enum Header {
    V1(Header0_1),
    V2(Header0_1),
    V3(v0_3::Header),
}

impl Header {
    pub fn height(&self) -> u64 {
        match self {
            Header::V1(header0_1) => header0_1.height,
            Header::V2(header0_1) => header0_1.height,
            Header::V3(header) => header.height,
        }
    }
}

impl<'de> Deserialize<'de> for Header {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: Deserializer<'de>,
    {
        struct HeaderVisitor;

        impl<'de> Visitor<'de> for HeaderVisitor {
            type Value = Header;

            fn expecting(&self, formatter: &mut fmt::Formatter) -> fmt::Result {
                formatter.write_str("Header")
            }

            fn visit_seq<V>(self, mut seq: V) -> Result<Self::Value, V::Error>
            where
                V: SeqAccess<'de>,
            {
                let chain_config_or_version: EitherOrVersion = seq
                    .next_element()?
                    .ok_or_else(|| de::Error::missing_field("chain_config"))?;

                match chain_config_or_version {
                    // For v0.1, the first field in the sequence of fields is the first field of the struct, so we call a function to get the rest of
                    // the fields from the sequence and pack them into the struct.
                    EitherOrVersion::Left(cfg) => Ok(Header::V1(
                        Header0_1::deserialize_with_chain_config(cfg.into(), seq)?,
                    )),
                    EitherOrVersion::Right(commit) => Ok(Header::V1(
                        Header0_1::deserialize_with_chain_config(commit.into(), seq)?,
                    )),
                    // For all versions > 0.1, the first "field" is not actually part of the `Header` struct.
                    // We just delegate directly to the derived deserialization impl for the appropriate version.
                    EitherOrVersion::Version(Version { major: 0, minor: 2 }) => Ok(Header::V2(
                        seq.next_element()?
                            .ok_or_else(|| de::Error::missing_field("fields"))?,
                    )),
                    EitherOrVersion::Version(Version { major: 0, minor: 3 }) => Ok(Header::V3(
                        seq.next_element()?
                            .ok_or_else(|| de::Error::missing_field("fields"))?,
                    )),
                    EitherOrVersion::Version(v) => {
                        Err(serde::de::Error::custom(format!("invalid version {v:?}")))
                    }
                }
            }

            fn visit_map<V>(self, mut map: V) -> Result<Header, V::Error>
            where
                V: MapAccess<'de>,
            {
                // insert all the fields in the serde_map as the map may have out of order fields.
                let mut serde_map: Map<String, Value> = Map::new();

                while let Some(key) = map.next_key::<String>()? {
                    serde_map.insert(key.trim().to_owned(), map.next_value()?);
                }

                if let Some(v) = serde_map.get("version") {
                    let fields = serde_map
                        .get("fields")
                        .ok_or_else(|| de::Error::missing_field("fields"))?;

                    let version = serde_json::from_value::<EitherOrVersion>(v.clone())
                        .map_err(de::Error::custom)?;
                    let result = match version {
                        EitherOrVersion::Version(Version { major: 0, minor: 2 }) => Ok(Header::V2(
                            serde_json::from_value(fields.clone()).map_err(de::Error::custom)?,
                        )),
                        EitherOrVersion::Version(Version { major: 0, minor: 3 }) => Ok(Header::V3(
                            serde_json::from_value(fields.clone()).map_err(de::Error::custom)?,
                        )),
                        EitherOrVersion::Version(v) => {
                            Err(de::Error::custom(format!("invalid version {v:?}")))
                        }
                        chain_config => Err(de::Error::custom(format!(
                            "expected version, found chain_config {chain_config:?}"
                        ))),
                    };
                    return result;
                }

                Ok(Header::V1(
                    serde_json::from_value(serde_map.into()).map_err(de::Error::custom)?,
                ))
            }
        }

        // List of all possible fields of all versions of the `Header`.
        // serde's `deserialize_struct` works by deserializing to a struct with a specific list of fields.
        // The length of the fields list we provide is always going to be greater than the length of the target struct.
        // In our case, we are deserializing to either a V1 Header or a VersionedHeader for versions > 0.1.
        // We use serde_json and bincode serialization in the sequencer.
        // Fortunately, serde_json ignores fields parameter and only cares about our Visitor implementation.
        // -  https://docs.rs/serde_json/1.0.120/serde_json/struct.Deserializer.html#method.deserialize_struct
        // Bincode uses the length of the fields list, but the bincode deserialization only cares that the length of the fields
        // is an upper bound of the target struct's fields length.
        // -  https://docs.rs/bincode/1.3.3/src/bincode/de/mod.rs.html#313
        // This works because the bincode deserializer only consumes the next field when `next_element` is called,
        // and our visitor calls it the correct number of times.
        // This would, however, break if the bincode deserializer implementation required an exact match of the field's length,
        // consuming one element for each field.
        let fields: &[&str] = &[
            "fields",
            "chain_config",
            "version",
            "height",
            "timestamp",
            "l1_head",
            "l1_finalized",
            "payload_commitment",
            "builder_commitment",
            "ns_table",
            "block_merkle_tree_root",
            "fee_merkle_tree_root",
            "fee_info",
            "builder_signature",
        ];

        deserializer.deserialize_struct("Header", fields, HeaderVisitor)
    }
}

impl Committable for Header {
    fn commit(&self) -> Commitment<Self> {
        match self {
            Self::V1(header) => header.commit(),
            Self::V2(fields) => RawCommitmentBuilder::new(&Self::tag())
                .u64_field("version_major", 0)
                .u64_field("version_minor", 2)
                .field("fields", fields.commit())
                .finalize(),
            Self::V3(fields) => RawCommitmentBuilder::new(&Self::tag())
                .u64_field("version_major", 0)
                .u64_field("version_minor", 3)
                .field("fields", fields.commit())
                .finalize(),
        }
    }

    fn tag() -> String {
        // We use the tag "BLOCK" since blocks are identified by the hash of their header. This will
        // thus be more intuitive to users than "HEADER".
        "BLOCK".into()
    }
}

impl Serialize for Header {
    fn serialize<S>(&self, serializer: S) -> Result<S::Ok, S::Error>
    where
        S: Serializer,
    {
        match self {
            Self::V1(header) => header.serialize(serializer),
            Self::V2(fields) => VersionedHeader {
                version: EitherOrVersion::Version(Version { major: 0, minor: 2 }),
                fields: fields.clone(),
            }
            .serialize(serializer),
            Self::V3(fields) => VersionedHeader {
                version: EitherOrVersion::Version(Version { major: 0, minor: 3 }),
                fields: fields.clone(),
            }
            .serialize(serializer),
        }
    }
}

#[derive(Debug, Deserialize, Serialize)]
pub struct VersionedHeader<Fields> {
    pub(crate) version: EitherOrVersion,
    pub(crate) fields: Fields,
}

#[derive(Deserialize, Serialize, Debug)]
pub enum EitherOrVersion {
    Left(ChainConfig),
    Right(Commitment<ChainConfig>),
    Version(Version),
}

// Header values
#[derive(Clone, Debug, Deserialize, Serialize, Hash, PartialEq, Eq)]
pub struct Header0_1 {
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

impl Header0_1 {
    fn commit(&self) -> Commitment<Header> {
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

impl Header0_1 {
    pub fn deserialize_with_chain_config<'de, A>(
        chain_config: ResolvableChainConfig,
        mut seq: A,
    ) -> Result<Self, A::Error>
    where
        A: SeqAccess<'de>,
    {
        macro_rules! element {
            ($seq:expr, $field:ident) => {
                $seq.next_element()?
                    .ok_or_else(|| de::Error::missing_field(stringify!($field)))?
            };
        }
        let height = element!(seq, height);
        let timestamp = element!(seq, timestamp);
        let l1_head = element!(seq, l1_head);
        let l1_finalized = element!(seq, l1_finalized);
        let payload_commitment = element!(seq, payload_commitment);
        let builder_commitment = element!(seq, builder_commitment);
        let ns_table = element!(seq, ns_table);
        let block_merkle_tree_root = element!(seq, block_merkle_tree_root);
        let fee_merkle_tree_root = element!(seq, fee_merkle_tree_root);
        let fee_info = element!(seq, fee_info);
        let builder_signature = element!(seq, builder_signature);

        Ok(Self {
            chain_config,
            height,
            timestamp,
            l1_head,
            l1_finalized,
            payload_commitment,
            builder_commitment,
            ns_table,
            block_merkle_tree_root,
            fee_merkle_tree_root,
            fee_info,
            builder_signature,
        })
    }
}

/// Type for protocol version number
#[derive(Deserialize, Serialize, Debug)]
pub struct Version {
    /// major version number
    pub major: u16,
    /// minor version number
    pub minor: u16,
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

    use super::Header0_1;

    #[test]
    fn header_test() {
        let header_str = include_str!("./mock_data/header.json");
        let header = serde_json::from_str::<Header0_1>(&header_str).unwrap();
        // Copied from espresso sequencer reference test
        let expected = "BLOCK~6Ol30XYkdKaNFXw0QAkcif18Lk8V8qkC4M81qTlwL707";
        assert_eq!(header.commit().to_string(), expected);
    }
}
