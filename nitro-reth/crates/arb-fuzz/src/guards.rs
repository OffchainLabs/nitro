use alloy_primitives::{Address, B256, U256};
use arb_test_harness::{
    node::{BlockId, ExecutionNode},
    scenario::{Scenario, ScenarioSetup, ScenarioStep, StateCheck},
};

use crate::shared_nodes::{fuzz_arbos_version, shared_dual_exec, FUZZ_L2_CHAIN_ID};

#[derive(Default)]
struct LastTxExpect {
    min_gas: Option<u64>,
    max_gas: Option<u64>,
    status: Option<bool>,
    log_count: Option<usize>,
}

#[derive(Default)]
struct SentinelExpect {
    address: Address,
    slot: B256,
    value: B256,
}

pub struct GuardedRun {
    name: String,
    steps: Vec<ScenarioStep>,
    state_checks: Vec<StateCheck>,
    last_tx: LastTxExpect,
    sentinels: Vec<SentinelExpect>,
    chain_id: u64,
    arbos_version: u64,
}

impl GuardedRun {
    pub fn new(name: impl Into<String>, steps: Vec<ScenarioStep>) -> Self {
        Self {
            name: name.into(),
            steps,
            state_checks: Vec::new(),
            last_tx: LastTxExpect::default(),
            sentinels: Vec::new(),
            chain_id: FUZZ_L2_CHAIN_ID,
            arbos_version: fuzz_arbos_version(),
        }
    }

    pub fn expect_last_tx_min_gas(mut self, gas: u64) -> Self {
        self.last_tx.min_gas = Some(gas);
        self
    }

    pub fn expect_last_tx_max_gas(mut self, gas: u64) -> Self {
        self.last_tx.max_gas = Some(gas);
        self
    }

    pub fn expect_last_tx_status(mut self, ok: bool) -> Self {
        self.last_tx.status = Some(ok);
        self
    }

    pub fn expect_last_tx_log_count(mut self, n: usize) -> Self {
        self.last_tx.log_count = Some(n);
        self
    }

    pub fn expect_sentinel(mut self, address: Address, slot: U256, value: B256) -> Self {
        let slot_b = B256::from(slot.to_be_bytes::<32>());
        self.sentinels.push(SentinelExpect {
            address,
            slot: slot_b,
            value,
        });
        self.state_checks.push(StateCheck {
            address,
            slots: vec![slot_b],
            check_balance: false,
            check_nonce: false,
            check_code: false,
        });
        self
    }

    pub fn diff_storage(mut self, address: Address, slots: Vec<U256>) -> Self {
        let slots_b: Vec<B256> = slots
            .into_iter()
            .map(|s| B256::from(s.to_be_bytes::<32>()))
            .collect();
        self.state_checks.push(StateCheck {
            address,
            slots: slots_b,
            check_balance: false,
            check_nonce: false,
            check_code: false,
        });
        self
    }

    pub fn diff_account(mut self, address: Address) -> Self {
        self.state_checks.push(StateCheck {
            address,
            slots: Vec::new(),
            check_balance: true,
            check_nonce: true,
            check_code: true,
        });
        self
    }

    pub fn run(self) {
        let scen = Scenario {
            name: self.name.clone(),
            description: format!("guarded: {}", self.name),
            setup: ScenarioSetup {
                l2_chain_id: self.chain_id,
                arbos_version: self.arbos_version,
                genesis: None,
            },
            steps: self.steps,
        };

        let nodes = shared_dual_exec();
        let mut nodes = nodes.lock().expect("dual-exec mutex poisoned");
        let report = nodes
            .run_with_state_checks(&scen, &self.state_checks)
            .expect("run scenario");

        if !report.is_clean() {
            dump_and_panic(&self.name, &report);
        }

        let last_tx_hash = locate_last_user_tx_hash(&*nodes);
        if let Some(hash) = last_tx_hash {
            let receipt = nodes.right.receipt(hash);
            if let Ok(rec) = receipt {
                if let Some(min) = self.last_tx.min_gas {
                    assert!(
                        rec.gas_used >= min,
                        "[{}] last tx gas_used {} below floor {} — likely false-friend (early revert or no-op dispatch)",
                        self.name, rec.gas_used, min,
                    );
                }
                if let Some(max) = self.last_tx.max_gas {
                    assert!(
                        rec.gas_used <= max,
                        "[{}] last tx gas_used {} above ceiling {}",
                        self.name, rec.gas_used, max,
                    );
                }
                if let Some(ok) = self.last_tx.status {
                    let expected = if ok { 1u8 } else { 0u8 };
                    assert_eq!(
                        rec.status, expected,
                        "[{}] last tx status mismatch (got {}, expected {})",
                        self.name, rec.status, expected,
                    );
                }
                if let Some(n) = self.last_tx.log_count {
                    assert_eq!(
                        rec.logs.len(),
                        n,
                        "[{}] last tx log count mismatch (got {}, expected {})",
                        self.name, rec.logs.len(), n,
                    );
                }
            }
        } else if self.last_tx.min_gas.is_some()
            || self.last_tx.status.is_some()
            || self.last_tx.log_count.is_some()
        {
            panic!(
                "[{}] no user tx found to assert last-tx expectations against",
                self.name
            );
        }

        for s in &self.sentinels {
            let got = nodes
                .right
                .storage(s.address, s.slot, BlockId::Latest)
                .unwrap_or(B256::ZERO);
            assert_eq!(
                got, s.value,
                "[{}] sentinel {} slot {:#x} mismatch (got {:#x}, expected {:#x}) — intended code path likely never ran",
                self.name, s.address, s.slot, got, s.value,
            );
        }
    }
}

fn locate_last_user_tx_hash<L: ExecutionNode, R: ExecutionNode>(
    nodes: &arb_test_harness::dual_exec::DualExec<L, R>,
) -> Option<B256> {
    let latest = nodes.right.block(BlockId::Latest).ok()?;
    for n in (0..=latest.number).rev() {
        let b = nodes.right.block(BlockId::Number(n)).ok()?;
        if let Some(h) = b.tx_hashes.iter().rev().find(|h| {
            nodes
                .right
                .receipt(**h)
                .map(|r| !is_internal_sender(r.from))
                .unwrap_or(false)
        }) {
            return Some(*h);
        }
    }
    None
}

fn is_internal_sender(addr: Address) -> bool {
    addr.0[0] == 0xa4 && addr.0[1] == 0xb0
}

fn dump_and_panic(name: &str, report: &arb_test_harness::dual_exec::DiffReport) {
    let payload = serde_json::json!({
        "scenario": name,
        "block_diffs": format!("{:#?}", report.block_diffs),
        "tx_diffs": format!("{:#?}", report.tx_diffs),
        "state_diffs": format!("{:#?}", report.state_diffs),
        "log_diffs": format!("{:#?}", report.log_diffs),
    });
    let path = std::path::PathBuf::from(format!("/tmp/guarded_{name}.json"));
    let _ = std::fs::write(&path, serde_json::to_vec_pretty(&payload).unwrap());
    panic!("arbreth diverged from Nitro on {name}; see {}", path.display());
}
