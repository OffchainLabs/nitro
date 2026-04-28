use std::path::{Path, PathBuf};

use anyhow::{Context, Result};
use serde::Serialize;
use serde_json::Value;

use arb_spec_tests::ExecutionFixture;
use arb_test_harness::{node::remote::RemoteNode, DiffReport, DualExec};

use super::TriageArgs;

#[derive(Debug, Clone, Serialize)]
pub struct TriageEntry {
    pub fixture: PathBuf,
    pub status: TriageStatus,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub diff: Option<DiffReport>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub error: Option<String>,
}

#[derive(Debug, Clone, Copy, Serialize)]
#[serde(rename_all = "lowercase")]
pub enum TriageStatus {
    Ok,
    Bug,
    Harness,
    #[allow(dead_code)]
    Approximation,
}

pub fn run(args: TriageArgs) -> Result<()> {
    let fixtures = collect_fixtures(&args.fixtures_dir)?;
    let mut report: Vec<TriageEntry> = Vec::with_capacity(fixtures.len());

    for path in fixtures {
        report.push(triage_one(&path, &args.nitro_rpc, &args.arbreth_rpc));
    }

    let body = serde_json::to_string_pretty(&Value::Array(
        report
            .into_iter()
            .map(|e| serde_json::to_value(e).unwrap_or(Value::Null))
            .collect(),
    ))
    .context("encode triage report")?;
    println!("{body}");
    Ok(())
}

fn triage_one(path: &Path, nitro_rpc: &str, arbreth_rpc: &str) -> TriageEntry {
    let fixture = match ExecutionFixture::load(path) {
        Ok(f) => f,
        Err(e) => {
            return TriageEntry {
                fixture: path.to_path_buf(),
                status: TriageStatus::Harness,
                diff: None,
                error: Some(format!("load: {e}")),
            };
        }
    };

    let scenario = match scenario_for(&fixture) {
        Ok(s) => s,
        Err(e) => {
            return TriageEntry {
                fixture: path.to_path_buf(),
                status: TriageStatus::Harness,
                diff: None,
                error: Some(format!("scenario: {e}")),
            };
        }
    };

    let left = RemoteNode::nitro(nitro_rpc);
    let right = RemoteNode::arbreth(arbreth_rpc);
    let mut dual = DualExec::new(left, right);
    let report = match dual.run(&scenario) {
        Ok(r) => r,
        Err(e) => {
            return TriageEntry {
                fixture: path.to_path_buf(),
                status: TriageStatus::Harness,
                diff: None,
                error: Some(format!("dual_exec: {e}")),
            };
        }
    };

    let status = classify(&report);
    TriageEntry {
        fixture: path.to_path_buf(),
        status,
        diff: Some(report),
        error: None,
    }
}

fn classify(report: &DiffReport) -> TriageStatus {
    if report.is_clean() {
        return TriageStatus::Ok;
    }
    let only_presence = report
        .block_diffs
        .iter()
        .all(|d| d.field == "presence")
        && report.tx_diffs.iter().all(|d| d.field == "presence")
        && report.log_diffs.iter().all(|d| d.field == "presence");
    if only_presence {
        TriageStatus::Harness
    } else {
        TriageStatus::Bug
    }
}

fn collect_fixtures(dir: &Path) -> Result<Vec<PathBuf>> {
    let mut out = Vec::new();
    walk(dir, &mut out)?;
    out.sort();
    Ok(out)
}

fn walk(dir: &Path, out: &mut Vec<PathBuf>) -> Result<()> {
    let entries = std::fs::read_dir(dir)
        .with_context(|| format!("read_dir {}", dir.display()))?;
    for entry in entries {
        let entry = entry.with_context(|| format!("iter {}", dir.display()))?;
        let path = entry.path();
        let ft = entry
            .file_type()
            .with_context(|| format!("file_type {}", path.display()))?;
        if ft.is_dir() {
            walk(&path, out)?;
        } else if ft.is_file() && path.extension().and_then(|s| s.to_str()) == Some("json") {
            out.push(path);
        }
    }
    Ok(())
}

fn scenario_for(fixture: &ExecutionFixture) -> Result<arb_test_harness::Scenario> {
    use arb_test_harness::{Scenario, ScenarioSetup, ScenarioStep};

    let mut steps: Vec<ScenarioStep> = Vec::with_capacity(fixture.messages.len());
    let mut next_idx = 1u64;
    for (i, msg) in fixture.messages.iter().enumerate() {
        let idx = msg.msg_idx.unwrap_or(next_idx);
        let parsed: arb_test_harness::L1Message = serde_json::from_value(msg.message.clone())
            .with_context(|| format!("message {i} (idx {idx}): decode L1Message"))?;
        steps.push(ScenarioStep::Message {
            idx,
            message: parsed,
            delayed_messages_read: msg.delayed_messages_read,
        });
        next_idx = idx + 1;
    }
    Ok(Scenario {
        name: fixture.name.clone(),
        description: fixture.description.clone(),
        setup: ScenarioSetup::default(),
        steps,
    })
}
