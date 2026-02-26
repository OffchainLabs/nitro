use core::ops::{Add, Sub};

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
}

/// Number of resource kinds.
pub const NUM_RESOURCE_KIND: usize = 6;

impl ResourceKind {
    pub const ALL: [ResourceKind; NUM_RESOURCE_KIND] = [
        ResourceKind::Unknown,
        ResourceKind::Computation,
        ResourceKind::HistoryGrowth,
        ResourceKind::StorageAccess,
        ResourceKind::StorageGrowth,
        ResourceKind::L1Calldata,
    ];

    pub fn from_u8(v: u8) -> Option<Self> {
        match v {
            0 => Some(Self::Unknown),
            1 => Some(Self::Computation),
            2 => Some(Self::HistoryGrowth),
            3 => Some(Self::StorageAccess),
            4 => Some(Self::StorageGrowth),
            5 => Some(Self::L1Calldata),
            _ => None,
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

    pub fn new(kind: ResourceKind, amount: u64) -> Self {
        let mut mg = Self::zero();
        mg.gas[kind as usize] = amount;
        mg.total = amount;
        mg
    }

    pub fn computation_gas(amount: u64) -> Self {
        Self::new(ResourceKind::Computation, amount)
    }

    pub fn l1_calldata_gas(amount: u64) -> Self {
        Self::new(ResourceKind::L1Calldata, amount)
    }

    /// Returns the gas amount for the specified resource kind.
    pub fn get(&self, kind: ResourceKind) -> u64 {
        self.gas[kind as usize]
    }

    /// Returns the total gas across all dimensions.
    pub fn total(&self) -> u64 {
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

    /// Returns a copy with the given resource kind set to amount.
    /// Returns (result, overflowed).
    pub fn with(mut self, kind: ResourceKind, amount: u64) -> (Self, bool) {
        let old = self.gas[kind as usize];
        self.gas[kind as usize] = amount;
        // Adjust total: subtract old, add new
        let (t1, o1) = self.total.overflowing_sub(old);
        let (t2, o2) = t1.overflowing_add(amount);
        self.total = t2;
        (self, o1 || o2)
    }

    /// Saturating add of another MultiGas into self.
    pub fn saturating_add_into(&mut self, other: MultiGas) {
        for i in 0..NUM_RESOURCE_KIND {
            self.gas[i] = self.gas[i].saturating_add(other.gas[i]);
        }
        self.total = self.total.saturating_add(other.total);
    }

    /// Saturating increment a single resource kind.
    pub fn saturating_increment_into(&mut self, kind: ResourceKind, amount: u64) {
        self.gas[kind as usize] = self.gas[kind as usize].saturating_add(amount);
        self.total = self.total.saturating_add(amount);
    }

    /// Saturating subtract of another MultiGas from self.
    pub fn saturating_sub_into(&mut self, other: MultiGas) {
        for i in 0..NUM_RESOURCE_KIND {
            self.gas[i] = self.gas[i].saturating_sub(other.gas[i]);
        }
        self.total = self.total.saturating_sub(other.total);
    }

    /// Adds refund amount.
    pub fn add_refund(&mut self, amount: u64) {
        self.refund = self.refund.saturating_add(amount);
    }

    /// Subtracts from refund.
    pub fn sub_refund(&mut self, amount: u64) {
        self.refund = self.refund.saturating_sub(amount);
    }

    /// Returns true if all dimensions are zero.
    pub fn is_zero(&self) -> bool {
        self.total == 0 && self.refund == 0
    }

    /// Constructs from pairs of (ResourceKind, amount).
    pub fn from_pairs(pairs: &[(ResourceKind, u64)]) -> Self {
        let mut mg = Self::zero();
        for &(kind, amount) in pairs {
            mg.gas[kind as usize] = amount;
            mg.total = mg.total.saturating_add(amount);
        }
        mg
    }
}

impl Add for MultiGas {
    type Output = Self;

    fn add(self, rhs: Self) -> Self {
        let mut result = self;
        result.saturating_add_into(rhs);
        result
    }
}

impl Sub for MultiGas {
    type Output = Self;

    fn sub(self, rhs: Self) -> Self {
        let mut result = self;
        result.saturating_sub_into(rhs);
        result
    }
}
