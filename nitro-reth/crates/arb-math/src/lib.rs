//! Numeric helpers shared by ArbOS components.
//!
//! The functions here are pure, allocation-free and operate on primitive
//! integers, so the crate is `no_std`-compatible by default.

#![cfg_attr(not(test), warn(unused_crate_dependencies))]
#![cfg_attr(not(feature = "std"), no_std)]

/// One unit expressed in basis points.
pub const ONE_IN_BIPS: u64 = 10_000;

/// Computes `b * e^(x / b)` where `b = 10_000` (basis points), using Horner's
/// rule applied to the Maclaurin expansion of `e^x` truncated at `accuracy`
/// terms.
///
/// The polynomial evaluated is
/// `b * (1 + x/b * (1 + x/(2b) * (1 + ... * (1 + x/(accuracy * b)))))`.
///
/// All intermediate operations saturate at [`u64::MAX`]; an `accuracy` of `0`
/// returns `b` unchanged (no polynomial terms beyond the constant).
///
/// # Examples
///
/// ```
/// use arb_math::{approx_exp_basis_points, ONE_IN_BIPS};
///
/// // exp(0) == 1.0, scaled by 10_000.
/// assert_eq!(approx_exp_basis_points(0, 12), ONE_IN_BIPS);
/// ```
pub fn approx_exp_basis_points(bips: u64, accuracy: u64) -> u64 {
    if bips == 0 || accuracy == 0 {
        return ONE_IN_BIPS;
    }

    let mut res = ONE_IN_BIPS.saturating_add(bips / accuracy);
    let mut i = accuracy - 1;
    while i > 0 {
        res = ONE_IN_BIPS.saturating_add(res.saturating_mul(bips) / (i * ONE_IN_BIPS));
        i -= 1;
    }
    res
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn zero_input_returns_one_bip() {
        for accuracy in [1u64, 4, 12, 32] {
            assert_eq!(approx_exp_basis_points(0, accuracy), ONE_IN_BIPS);
        }
    }

    #[test]
    fn zero_accuracy_returns_one_bip() {
        assert_eq!(approx_exp_basis_points(1234, 0), ONE_IN_BIPS);
        assert_eq!(approx_exp_basis_points(u64::MAX, 0), ONE_IN_BIPS);
    }

    /// Known reference values for accuracy = 4 (quartic Maclaurin).
    #[test]
    fn reference_values_accuracy_4() {
        let cases: &[(u64, u64)] = &[
            (0, 10_000),
            (500, 10_512),
            (2_000, 12_214),
            (10_000, 27_083),
            (20_000, 70_000),
        ];
        for &(input, expected) in cases {
            assert_eq!(
                approx_exp_basis_points(input, 4),
                expected,
                "accuracy=4 input={input}",
            );
        }
    }

    /// Known reference values for accuracy = 12 (degree-12 Maclaurin).
    #[test]
    fn reference_values_accuracy_12() {
        let cases: &[(u64, u64)] = &[
            (0, 10_000),
            (500, 10_512),
            (2_000, 12_214),
            (10_000, 27_182),
            (20_000, 73_888),
            (30_000, 200_848),
            (50_000, 1_481_075),
            (100_000, 174_348_300),
        ];
        for &(input, expected) in cases {
            assert_eq!(
                approx_exp_basis_points(input, 12),
                expected,
                "accuracy=12 input={input}",
            );
        }
    }

    #[test]
    fn monotonically_non_decreasing_accuracy_12() {
        let mut prev = approx_exp_basis_points(0, 12);
        for x in (0..100_000).step_by(137) {
            let cur = approx_exp_basis_points(x, 12);
            assert!(
                cur >= prev,
                "non-monotonic at x={x}: prev={prev}, cur={cur}"
            );
            prev = cur;
        }
    }

    /// Pins the divergence point between Horner and Taylor at accuracy 12. The
    /// quick-and-dirty Taylor series returns 1_481_128 for this input; the
    /// Horner-evaluated polynomial returns 1_481_075.
    #[test]
    fn regression_horner_not_taylor_at_50000() {
        assert_eq!(approx_exp_basis_points(50_000, 12), 1_481_075);
    }

    #[test]
    fn saturates_on_huge_input() {
        let v = approx_exp_basis_points(u64::MAX, 12);
        assert!(v > 0, "huge input must not return zero");
    }
}
