use std::path::Path;

use anyhow::{Context, Result};

use arb_spec_tests::{ExecutionExpectations, ExecutionFixture};
use arb_test_harness::{capture::capture_from_node, node::remote::RemoteNode};

use super::RecordArgs;

pub fn run(args: RecordArgs) -> Result<()> {
    let mut fixture = ExecutionFixture::load(&args.fixture)
        .with_context(|| format!("load fixture {}", args.fixture.display()))?;

    let scenario = scenario_for(&fixture)?;
    let mut node = RemoteNode::nitro(args.nitro_rpc.as_str());
    let captured = capture_from_node(&mut node, &scenario)
        .with_context(|| format!("capture from {}", args.nitro_rpc))?;

    let parsed: ExecutionExpectations =
        serde_json::from_value(captured.expected_json).context("decode captured expectations")?;
    fixture.expected = parsed;

    let target: &Path = args.out.as_deref().unwrap_or(&args.fixture);
    let body = serde_json::to_vec_pretty(&fixture).context("serialize fixture")?;
    std::fs::write(target, body).with_context(|| format!("write {}", target.display()))?;

    println!("recorded {} -> {}", args.fixture.display(), target.display());
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
