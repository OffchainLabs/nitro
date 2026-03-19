/// Memory pricing model for Stylus WASM programs.
#[derive(Debug, Clone, Copy)]
pub struct MemoryModel {
    pub free_pages: u16,
    pub page_gas: u16,
}

impl MemoryModel {
    pub fn new(free_pages: u16, page_gas: u16) -> Self {
        Self {
            free_pages,
            page_gas,
        }
    }

    /// Gas cost of allocating `new_pages` given `open` active and `ever` ever used.
    pub fn gas_cost(&self, new_pages: u16, open: u16, ever: u16) -> u64 {
        let new_open = open.saturating_add(new_pages);
        let new_ever = ever.max(new_open);

        if new_ever <= self.free_pages {
            return 0;
        }

        let sub_free = |pages: u16| -> u16 { pages.saturating_sub(self.free_pages) };

        let adding = sub_free(new_open).saturating_sub(sub_free(open));
        let linear = (adding as u64).saturating_mul(self.page_gas as u64);
        let expand = self.exp(new_ever).saturating_sub(self.exp(ever));
        linear.saturating_add(expand)
    }

    fn exp(&self, pages: u16) -> u64 {
        let idx = pages as usize;
        if idx < MEMORY_EXPONENTS.len() {
            MEMORY_EXPONENTS[idx] as u64
        } else {
            u64::MAX
        }
    }
}

/// Precomputed exponential growth table for memory costs.
static MEMORY_EXPONENTS: [u32; 129] = [
    1, 1, 1, 1, 1, 1, 2, 2, 2, 3, 3, 4, 5, 5, 6, 7, 8, 9, 11, 12, 14, 17, 19, 22, 25, 29, 33, 38,
    43, 50, 57, 65, 75, 85, 98, 112, 128, 147, 168, 193, 221, 253, 289, 331, 379, 434, 497, 569,
    651, 745, 853, 976, 1117, 1279, 1463, 1675, 1917, 2194, 2511, 2874, 3290, 3765, 4309, 4932,
    5645, 6461, 7395, 8464, 9687, 11087, 12689, 14523, 16621, 19024, 21773, 24919, 28521, 32642,
    37359, 42758, 48938, 56010, 64104, 73368, 83971, 96106, 109994, 125890, 144082, 164904, 188735,
    216010, 247226, 282953, 323844, 370643, 424206, 485509, 555672, 635973, 727880, 833067, 953456,
    1091243, 1248941, 1429429, 1636000, 1872423, 2143012, 2452704, 2807151, 3212820, 3677113,
    4208502, 4816684, 5512756, 6309419, 7221210, 8264766, 9459129, 10826093, 12390601, 14181199,
    16230562, 18576084, 21260563, 24332984, 27849408, 31873999,
];
