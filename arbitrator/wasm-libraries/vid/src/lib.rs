use ark_bls12_381::Bls12_381;
use go_abi::*;
use jf_primitives::{
    pcs::{checked_fft_size, prelude::UnivariateKzgPCS, PolynomialCommitmentScheme},
    vid::advz::{payload_prover::LargeRangeProof, Advz},
};

#[no_mangle]
pub unsafe extern "C" fn go__github_com_offchainlabs_nitro_arbvid_verifyNamespace(sp: GoStack) {
    // TODO implement: https://github.com/EspressoSystems/nitro-espresso-integration/issues/65
}
