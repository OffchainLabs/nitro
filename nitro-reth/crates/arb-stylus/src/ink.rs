/// Gas unit for EVM gas accounting.
#[derive(Clone, Copy, PartialEq, Eq, PartialOrd, Ord, Hash, Debug, Default)]
#[must_use]
pub struct Gas(pub u64);

/// Ink unit for Stylus computation metering.
/// 1 EVM gas = ink_price ink (default 10,000).
#[derive(Clone, Copy, PartialEq, Eq, PartialOrd, Ord, Hash, Debug, Default)]
#[must_use]
pub struct Ink(pub u64);

macro_rules! impl_math {
    ($t:ident) => {
        impl std::ops::Add for $t {
            type Output = Self;
            fn add(self, rhs: Self) -> Self {
                Self(self.0 + rhs.0)
            }
        }

        impl std::ops::AddAssign for $t {
            fn add_assign(&mut self, rhs: Self) {
                self.0 += rhs.0;
            }
        }

        impl std::ops::Sub for $t {
            type Output = Self;
            fn sub(self, rhs: Self) -> Self {
                Self(self.0 - rhs.0)
            }
        }

        impl std::ops::SubAssign for $t {
            fn sub_assign(&mut self, rhs: Self) {
                self.0 -= rhs.0;
            }
        }

        impl std::ops::Mul<u64> for $t {
            type Output = Self;
            fn mul(self, rhs: u64) -> Self {
                Self(self.0 * rhs)
            }
        }

        impl $t {
            pub const fn saturating_add(self, rhs: Self) -> Self {
                Self(self.0.saturating_add(rhs.0))
            }

            pub const fn saturating_sub(self, rhs: Self) -> Self {
                Self(self.0.saturating_sub(rhs.0))
            }

            pub fn to_be_bytes(self) -> [u8; 8] {
                self.0.to_be_bytes()
            }
        }
    };
}

impl_math!(Gas);
impl_math!(Ink);
