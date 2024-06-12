use std::ops::Range;

use ark_bn254::Bn254;
use ark_serialize::CanonicalDeserialize;
use jf_pcs::{
    prelude::UnivariateUniversalParams, univariate_kzg::UnivariateKzgPCS,
    PolynomialCommitmentScheme,
};
use jf_vid::advz::payload_prover::{LargeRangeProof, SmallRangeProof};
use jf_vid::{
    advz,
    payload_prover::{PayloadProver, Statement},
    precomputable::Precomputable,
    VidDisperse, VidResult, VidScheme,
};
use lazy_static::lazy_static;
use serde::{Deserialize, Serialize};
use sha2::Sha256;

/// Private type alias for the EC pairing type parameter for [`Advz`].
type E = Bn254;
/// Private type alias for the hash type parameter for [`Advz`].
type H = Sha256;

type Advz = advz::Advz<E, H>;

pub type VidCommitment = <VidSchemeType as VidScheme>::Commit;
pub type VidCommon = <VidSchemeType as VidScheme>::Common;

pub struct VidSchemeType(Advz);
pub const SRS_DEGREE: usize = 2u64.pow(20) as usize + 2;

impl VidScheme for VidSchemeType {
    type Commit = <Advz as VidScheme>::Commit;
    type Share = <Advz as VidScheme>::Share;
    type Common = <Advz as VidScheme>::Common;

    fn commit_only<B>(&mut self, payload: B) -> VidResult<Self::Commit>
    where
        B: AsRef<[u8]>,
    {
        self.0.commit_only(payload)
    }

    fn disperse<B>(&mut self, payload: B) -> VidResult<VidDisperse<Self>>
    where
        B: AsRef<[u8]>,
    {
        self.0.disperse(payload).map(vid_disperse_conversion)
    }

    fn verify_share(
        &self,
        share: &Self::Share,
        common: &Self::Common,
        commit: &Self::Commit,
    ) -> VidResult<Result<(), ()>> {
        self.0.verify_share(share, common, commit)
    }

    fn recover_payload(&self, shares: &[Self::Share], common: &Self::Common) -> VidResult<Vec<u8>> {
        self.0.recover_payload(shares, common)
    }

    fn is_consistent(commit: &Self::Commit, common: &Self::Common) -> VidResult<()> {
        <Advz as VidScheme>::is_consistent(commit, common)
    }

    fn get_payload_byte_len(common: &Self::Common) -> u32 {
        <Advz as VidScheme>::get_payload_byte_len(common)
    }

    fn get_num_storage_nodes(common: &Self::Common) -> u32 {
        <Advz as VidScheme>::get_num_storage_nodes(common)
    }

    fn get_multiplicity(common: &Self::Common) -> u32 {
        <Advz as VidScheme>::get_multiplicity(common)
    }
}

impl PayloadProver<SmallRangeProofType> for VidSchemeType {
    fn payload_proof<B>(&self, payload: B, range: Range<usize>) -> VidResult<SmallRangeProofType>
    where
        B: AsRef<[u8]>,
    {
        self.0
            .payload_proof(payload, range)
            .map(SmallRangeProofType)
    }

    fn payload_verify(
        &self,
        stmt: Statement<'_, Self>,
        proof: &SmallRangeProofType,
    ) -> VidResult<Result<(), ()>> {
        self.0.payload_verify(stmt_conversion(stmt), &proof.0)
    }
}

impl PayloadProver<LargeRangeProofType> for VidSchemeType {
    fn payload_proof<B>(&self, payload: B, range: Range<usize>) -> VidResult<LargeRangeProofType>
    where
        B: AsRef<[u8]>,
    {
        self.0
            .payload_proof(payload, range)
            .map(LargeRangeProofType)
    }

    fn payload_verify(
        &self,
        stmt: Statement<'_, Self>,
        proof: &LargeRangeProofType,
    ) -> VidResult<Result<(), ()>> {
        self.0.payload_verify(stmt_conversion(stmt), &proof.0)
    }
}

impl Precomputable for VidSchemeType {
    type PrecomputeData = <Advz as Precomputable>::PrecomputeData;

    fn commit_only_precompute<B>(
        &self,
        payload: B,
    ) -> VidResult<(Self::Commit, Self::PrecomputeData)>
    where
        B: AsRef<[u8]>,
    {
        self.0.commit_only_precompute(payload)
    }

    fn disperse_precompute<B>(
        &self,
        payload: B,
        data: &Self::PrecomputeData,
    ) -> VidResult<VidDisperse<Self>>
    where
        B: AsRef<[u8]>,
    {
        self.0
            .disperse_precompute(payload, data)
            .map(vid_disperse_conversion)
    }
}

lazy_static! {
    // Initialize the byte array from JSON content
    pub static ref KZG_SRS: UnivariateUniversalParams<Bn254> = {
        let json_content = include_str!("../vid_srs.json");
        let s: Vec<u8> = serde_json::from_str(json_content).expect("Failed to deserialize");
        UnivariateUniversalParams::<Bn254>::deserialize_uncompressed_unchecked(s.as_slice())
        .unwrap()
    };
}

#[derive(Clone, Debug, Eq, PartialEq, Serialize, Deserialize)]
pub struct LargeRangeProofType(
    // # Type complexity
    //
    // Jellyfish's `LargeRangeProof` type has a prime field generic parameter `F`.
    // This `F` is determined by the type parameter `E` for `Advz`.
    // Jellyfish needs a more ergonomic way for downstream users to refer to this type.
    //
    // There is a `KzgEval` type alias in jellyfish that helps a little, but it's currently private:
    // <https://github.com/EspressoSystems/jellyfish/issues/423>
    // If it were public then we could instead use
    // `LargeRangeProof<KzgEval<E>>`
    // but that's still pretty crufty.
    LargeRangeProof<<UnivariateKzgPCS<E> as PolynomialCommitmentScheme>::Evaluation>,
);

/// Newtype wrapper for a small payload range proof.
///
/// Useful for transaction proofs.
#[derive(Clone, Debug, Eq, PartialEq, Serialize, Deserialize)]
pub struct SmallRangeProofType(
    // # Type complexity
    //
    // Similar to the comments in `LargeRangeProofType`.
    SmallRangeProof<<UnivariateKzgPCS<E> as PolynomialCommitmentScheme>::Proof>,
);

#[must_use]
pub fn vid_scheme(num_storage_nodes: usize) -> VidSchemeType {
    // recovery_threshold is currently num_storage_nodes rounded down to a power of two
    // TODO recovery_threshold should be a function of the desired erasure code rate
    // https://github.com/EspressoSystems/HotShot/issues/2152
    let recovery_threshold = 1 << num_storage_nodes.ilog2();

    #[allow(clippy::panic)]
    let num_storage_nodes = u32::try_from(num_storage_nodes).unwrap_or_else(|err| {
        panic!(
            "num_storage_nodes {num_storage_nodes} should fit into u32; \
                error: {err}"
        )
    });

    // TODO panic, return `Result`, or make `new` infallible upstream (eg. by panicking)?
    #[allow(clippy::panic)]
    VidSchemeType(
        Advz::new(num_storage_nodes, recovery_threshold, &*KZG_SRS).unwrap_or_else(|err| {
              panic!("advz construction failure: (num_storage nodes,recovery_threshold)=({num_storage_nodes},{recovery_threshold}); \
                      error: {err}")
        })
    )
}

fn stmt_conversion(stmt: Statement<'_, VidSchemeType>) -> Statement<'_, Advz> {
    Statement {
        payload_subslice: stmt.payload_subslice,
        range: stmt.range,
        commit: stmt.commit,
        common: stmt.common,
    }
}

fn vid_disperse_conversion(vid_disperse: VidDisperse<Advz>) -> VidDisperse<VidSchemeType> {
    VidDisperse {
        shares: vid_disperse.shares,
        common: vid_disperse.common,
        commit: vid_disperse.commit,
    }
}
