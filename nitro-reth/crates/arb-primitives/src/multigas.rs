use core::fmt;
use core::ops::{Add, Sub};
use serde::{Deserialize, Serialize};

/// Resource kinds for multi-dimensional gas metering.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
#[repr(u8)]
pub enum ResourceKind {
    Unknown = 0,
    Computation = 1,
    HistoryGrowth = 2,
    StorageAccess = 3,
    StorageGrowth = 4,
    L1Calldata = 5,
    L2Calldata = 6,
    WasmComputation = 7,
}

/// Number of resource kinds.
pub const NUM_RESOURCE_KIND: usize = 8;

impl ResourceKind {
    pub const ALL: [ResourceKind; NUM_RESOURCE_KIND] = [
        ResourceKind::Unknown,
        ResourceKind::Computation,
        ResourceKind::HistoryGrowth,
        ResourceKind::StorageAccess,
        ResourceKind::StorageGrowth,
        ResourceKind::L1Calldata,
        ResourceKind::L2Calldata,
        ResourceKind::WasmComputation,
    ];

    pub fn from_u8(v: u8) -> Option<Self> {
        match v {
            0 => Some(Self::Unknown),
            1 => Some(Self::Computation),
            2 => Some(Self::HistoryGrowth),
            3 => Some(Self::StorageAccess),
            4 => Some(Self::StorageGrowth),
            5 => Some(Self::L1Calldata),
            6 => Some(Self::L2Calldata),
            7 => Some(Self::WasmComputation),
            _ => None,
        }
    }
}

impl fmt::Display for ResourceKind {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::Unknown => write!(f, "Unknown"),
            Self::Computation => write!(f, "Computation"),
            Self::HistoryGrowth => write!(f, "HistoryGrowth"),
            Self::StorageAccess => write!(f, "StorageAccess"),
            Self::StorageGrowth => write!(f, "StorageGrowth"),
            Self::L1Calldata => write!(f, "L1Calldata"),
            Self::L2Calldata => write!(f, "L2Calldata"),
            Self::WasmComputation => write!(f, "WasmComputation"),
        }
    }
}

/// Multi-dimensional gas tracking.
///
/// Tracks gas usage per resource kind, a total, and a refund amount.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub struct MultiGas {
    gas: [u64; NUM_RESOURCE_KIND],
    total: u64,
    refund: u64,
}

impl MultiGas {
    pub const fn zero() -> Self {
        Self {
            gas: [0; NUM_RESOURCE_KIND],
            total: 0,
            refund: 0,
        }
    }

    /// Construct from raw arrays (used in deserialization).
    pub const fn from_raw(gas: [u64; NUM_RESOURCE_KIND], total: u64, refund: u64) -> Self {
        Self { gas, total, refund }
    }

    pub fn new(kind: ResourceKind, amount: u64) -> Self {
        let mut mg = Self::zero();
        mg.gas[kind as usize] = amount;
        mg.total = amount;
        mg
    }

    /// Constructs from pairs of (ResourceKind, amount). Panics on overflow.
    pub fn from_pairs(pairs: &[(ResourceKind, u64)]) -> Self {
        let mut mg = Self::zero();
        for &(kind, amount) in pairs {
            mg.gas[kind as usize] = amount;
            mg.total = mg
                .total
                .checked_add(amount)
                .expect("multigas overflow");
        }
        mg
    }

    pub fn unknown_gas(amount: u64) -> Self {
        Self::new(ResourceKind::Unknown, amount)
    }

    pub fn computation_gas(amount: u64) -> Self {
        Self::new(ResourceKind::Computation, amount)
    }

    pub fn history_growth_gas(amount: u64) -> Self {
        Self::new(ResourceKind::HistoryGrowth, amount)
    }

    pub fn storage_access_gas(amount: u64) -> Self {
        Self::new(ResourceKind::StorageAccess, amount)
    }

    pub fn storage_growth_gas(amount: u64) -> Self {
        Self::new(ResourceKind::StorageGrowth, amount)
    }

    pub fn l1_calldata_gas(amount: u64) -> Self {
        Self::new(ResourceKind::L1Calldata, amount)
    }

    pub fn l2_calldata_gas(amount: u64) -> Self {
        Self::new(ResourceKind::L2Calldata, amount)
    }

    pub fn wasm_computation_gas(amount: u64) -> Self {
        Self::new(ResourceKind::WasmComputation, amount)
    }

    /// Returns the gas amount for the specified resource kind.
    pub fn get(&self, kind: ResourceKind) -> u64 {
        self.gas[kind as usize]
    }

    /// Returns a copy with the given resource kind set to amount.
    /// Returns (result, overflowed).
    pub fn with(self, kind: ResourceKind, amount: u64) -> (Self, bool) {
        let mut res = self;
        let old = res.gas[kind as usize];
        match (res.total - old).checked_add(amount) {
            Some(new_total) => {
                res.gas[kind as usize] = amount;
                res.total = new_total;
                (res, false)
            }
            None => (self, true),
        }
    }

    /// Returns the total gas across all dimensions.
    pub fn total(&self) -> u64 {
        self.total
    }

    /// Alias matching Go's SingleGas().
    pub fn single_gas(&self) -> u64 {
        self.total
    }

    /// Returns the refund amount.
    pub fn refund(&self) -> u64 {
        self.refund
    }

    /// Returns a copy with the refund set.
    pub fn with_refund(mut self, refund: u64) -> Self {
        self.refund = refund;
        self
    }

    /// Checked add. Returns (result, overflowed).
    pub fn safe_add(self, x: MultiGas) -> (Self, bool) {
        let mut res = self;
        for i in 0..NUM_RESOURCE_KIND {
            match res.gas[i].checked_add(x.gas[i]) {
                Some(v) => res.gas[i] = v,
                None => return (self, true),
            }
        }
        match res.total.checked_add(x.total) {
            Some(t) => res.total = t,
            None => return (self, true),
        }
        match res.refund.checked_add(x.refund) {
            Some(r) => res.refund = r,
            None => return (self, true),
        }
        (res, false)
    }

    /// Saturating add, returning a new value.
    pub fn saturating_add(self, x: MultiGas) -> Self {
        let mut res = self;
        for i in 0..NUM_RESOURCE_KIND {
            res.gas[i] = res.gas[i].saturating_add(x.gas[i]);
        }
        res.total = res.total.saturating_add(x.total);
        res.refund = res.refund.saturating_add(x.refund);
        res
    }

    /// Saturating add of another MultiGas into self (in-place).
    pub fn saturating_add_into(&mut self, other: MultiGas) {
        for i in 0..NUM_RESOURCE_KIND {
            self.gas[i] = self.gas[i].saturating_add(other.gas[i]);
        }
        self.total = self.total.saturating_add(other.total);
        self.refund = self.refund.saturating_add(other.refund);
    }

    /// Checked subtract. Returns (result, underflowed).
    pub fn safe_sub(self, x: MultiGas) -> (Self, bool) {
        let mut res = self;
        for i in 0..NUM_RESOURCE_KIND {
            match res.gas[i].checked_sub(x.gas[i]) {
                Some(v) => res.gas[i] = v,
                None => return (self, true),
            }
        }
        match res.total.checked_sub(x.total) {
            Some(t) => res.total = t,
            None => return (self, true),
        }
        match res.refund.checked_sub(x.refund) {
            Some(r) => res.refund = r,
            None => return (self, true),
        }
        (res, false)
    }

    /// Saturating subtract, returning a new value.
    pub fn saturating_sub(self, x: MultiGas) -> Self {
        let mut res = self;
        for i in 0..NUM_RESOURCE_KIND {
            res.gas[i] = res.gas[i].saturating_sub(x.gas[i]);
        }
        res.total = res.total.saturating_sub(x.total);
        res.refund = res.refund.saturating_sub(x.refund);
        res
    }

    /// Saturating subtract in place.
    pub fn saturating_sub_into(&mut self, other: MultiGas) {
        for i in 0..NUM_RESOURCE_KIND {
            self.gas[i] = self.gas[i].saturating_sub(other.gas[i]);
        }
        self.total = self.total.saturating_sub(other.total);
        self.refund = self.refund.saturating_sub(other.refund);
    }

    /// Checked increment of a single resource kind and total.
    /// Returns (result, overflowed).
    pub fn safe_increment(self, kind: ResourceKind, gas: u64) -> (Self, bool) {
        let mut res = self;
        match res.gas[kind as usize].checked_add(gas) {
            Some(v) => res.gas[kind as usize] = v,
            None => return (self, true),
        }
        match res.total.checked_add(gas) {
            Some(t) => res.total = t,
            None => return (self, true),
        }
        (res, false)
    }

    /// Saturating increment of a single resource kind and total.
    pub fn saturating_increment(self, kind: ResourceKind, gas: u64) -> Self {
        let mut res = self;
        res.gas[kind as usize] = res.gas[kind as usize].saturating_add(gas);
        res.total = res.total.saturating_add(gas);
        res
    }

    /// Saturating increment in place (hot-path variant).
    pub fn saturating_increment_into(&mut self, kind: ResourceKind, amount: u64) {
        self.gas[kind as usize] = self.gas[kind as usize].saturating_add(amount);
        self.total = self.total.saturating_add(amount);
    }

    /// Adds refund amount (saturating).
    pub fn add_refund(&mut self, amount: u64) {
        self.refund = self.refund.saturating_add(amount);
    }

    /// Subtracts from refund (saturating).
    pub fn sub_refund(&mut self, amount: u64) {
        self.refund = self.refund.saturating_sub(amount);
    }

    /// Returns true if all fields are zero.
    pub fn is_zero(&self) -> bool {
        self.total == 0 && self.refund == 0 && self.gas == [0u64; NUM_RESOURCE_KIND]
    }
}

impl Add for MultiGas {
    type Output = Self;

    fn add(self, rhs: Self) -> Self {
        self.saturating_add(rhs)
    }
}

impl Sub for MultiGas {
    type Output = Self;

    fn sub(self, rhs: Self) -> Self {
        self.saturating_sub(rhs)
    }
}

impl Serialize for MultiGas {
    fn serialize<S: serde::Serializer>(&self, serializer: S) -> Result<S::Ok, S::Error> {
        use serde::ser::SerializeStruct;
        let mut s = serializer.serialize_struct("MultiGas", 10)?;
        s.serialize_field("unknown", &format!("{:#x}", self.gas[0]))?;
        s.serialize_field("computation", &format!("{:#x}", self.gas[1]))?;
        s.serialize_field("historyGrowth", &format!("{:#x}", self.gas[2]))?;
        s.serialize_field("storageAccess", &format!("{:#x}", self.gas[3]))?;
        s.serialize_field("storageGrowth", &format!("{:#x}", self.gas[4]))?;
        s.serialize_field("l1Calldata", &format!("{:#x}", self.gas[5]))?;
        s.serialize_field("l2Calldata", &format!("{:#x}", self.gas[6]))?;
        s.serialize_field("wasmComputation", &format!("{:#x}", self.gas[7]))?;
        s.serialize_field("refund", &format!("{:#x}", self.refund))?;
        s.serialize_field("total", &format!("{:#x}", self.total))?;
        s.end()
    }
}

impl<'de> Deserialize<'de> for MultiGas {
    fn deserialize<D: serde::Deserializer<'de>>(deserializer: D) -> Result<Self, D::Error> {
        #[derive(Deserialize)]
        #[serde(rename_all = "camelCase")]
        struct Helper {
            #[serde(default)]
            unknown: HexU64,
            #[serde(default)]
            computation: HexU64,
            #[serde(default)]
            history_growth: HexU64,
            #[serde(default)]
            storage_access: HexU64,
            #[serde(default)]
            storage_growth: HexU64,
            #[serde(default)]
            l1_calldata: HexU64,
            #[serde(default)]
            l2_calldata: HexU64,
            #[serde(default)]
            wasm_computation: HexU64,
            #[serde(default)]
            refund: HexU64,
            #[serde(default)]
            total: HexU64,
        }

        let h = Helper::deserialize(deserializer)?;
        Ok(MultiGas {
            gas: [
                h.unknown.0,
                h.computation.0,
                h.history_growth.0,
                h.storage_access.0,
                h.storage_growth.0,
                h.l1_calldata.0,
                h.l2_calldata.0,
                h.wasm_computation.0,
            ],
            refund: h.refund.0,
            total: h.total.0,
        })
    }
}

/// Helper for hex-encoded u64 deserialization.
#[derive(Default)]
struct HexU64(u64);

impl<'de> Deserialize<'de> for HexU64 {
    fn deserialize<D: serde::Deserializer<'de>>(deserializer: D) -> Result<Self, D::Error> {
        let s: String = String::deserialize(deserializer)?;
        let v = u64::from_str_radix(s.trim_start_matches("0x"), 16)
            .map_err(serde::de::Error::custom)?;
        Ok(HexU64(v))
    }
}
