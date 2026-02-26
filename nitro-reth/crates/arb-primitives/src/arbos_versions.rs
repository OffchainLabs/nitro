/// ArbOS version identifiers.
///
/// Controls version-gated behavior across the node.
#[derive(Debug, Clone, Copy, PartialEq, Eq, PartialOrd, Ord, Hash)]
#[repr(u64)]
pub enum ArbOSVersion {
    V1 = 1,
    V2 = 2,
    V3 = 3,
    V4 = 4,
    V5 = 5,
    V10 = 10,
    V11 = 11,
    V20 = 20,
    V30 = 30,
    V31 = 31,
    V32 = 32,
    V40 = 40,
}

impl ArbOSVersion {
    pub fn from_u64(v: u64) -> Option<Self> {
        match v {
            1 => Some(Self::V1),
            2 => Some(Self::V2),
            3 => Some(Self::V3),
            4 => Some(Self::V4),
            5 => Some(Self::V5),
            10 => Some(Self::V10),
            11 => Some(Self::V11),
            20 => Some(Self::V20),
            30 => Some(Self::V30),
            31 => Some(Self::V31),
            32 => Some(Self::V32),
            40 => Some(Self::V40),
            _ => None,
        }
    }

    pub fn as_u64(self) -> u64 {
        self as u64
    }
}
