use anyhow::{Context, Result};

use arb_spec_tests::ExecutionFixture;
use arb_test_harness::{node::remote::RemoteNode, DualExec};

use crate::cli::CompareArgs;

pub fn run(args: CompareArgs) -> Result<()> {
    let fixture = ExecutionFixture::load(&args.fixture)
        .with_context(|| format!("load fixture {}", args.fixture.display()))?;

    let scenario = scenario_for(&fixture)?;
    let left = RemoteNode::nitro(args.nitro_rpc.as_str());
    let right = RemoteNode::arbreth(args.arbreth_rpc.as_str());
    let mut dual = DualExec::new(left, right);
    let report = dual
        .run(&scenario)
        .map_err(|e| anyhow::anyhow!("dual_exec: {e}"))?;

    let json = serde_json::to_string_pretty(&report).context("encode diff report")?;
    println!("{json}");
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
