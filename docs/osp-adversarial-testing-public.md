# Adversarial Testing BOLD OSP: Deterministic Host I/O Divergence (Public Summary)

## TL;DR

We built an **adversarial testing harness** for Arbitrum Nitro’s BOLD challenge protocol that intentionally introduces a **deterministic divergence in Host I/O behavior** between an “honest” node and a “fault-injected” node. We then verified that:

- The disagreement can be narrowed down through BOLD’s bisection stages, and
- The dispute can reach **One-Step Proof (OSP)** on L1, where the on-chain prover acts as the final source of truth.

This document focuses on the **why** and **what we learned**, and intentionally omits operational details that could be misused.

## Why we did this (product + security perspective)

Fraud proof systems must work in the real world: bugs, misconfigurations, partial upgrades, and heterogeneous environments happen. When a dispute occurs, we need high confidence that:

1. **Disagreements are resolvable** (the protocol converges toward a minimal point of contention),
2. **L1 adjudication is authoritative** (the “true” semantics are enforced on-chain), and
3. The system remains operable through the entire dispute lifecycle, including the final OSP step.

Adversarial testing helps validate these properties under realistic conditions. It is not only a security exercise—it is also a robustness and operational readiness exercise.

## Background (for broader audiences)

### What is BOLD?

BOLD is Arbitrum’s interactive dispute protocol design for narrowing down disagreements. The idea is to progressively bisect the contested execution until it can be resolved at the smallest meaningful granularity.

### What is One-Step Proof (OSP)?

OSP is the final dispute resolution step on L1: once the dispute is narrowed to a single step of the VM, a prover contract executes **one step** on-chain and verifies which party is correct.

In short:

- Off-chain participants can be wrong or buggy.
- The final arbiter is **the L1 prover**, which is deterministic and consensus-driven.

## Our approach: deterministic Host I/O divergence

### Why Host I/O?

The Nitro VM interacts with its environment (e.g., inbox data, preimages, global state) through **Host I/O**. These operations are especially important because:

- They bridge “pure computation” and “external inputs”.
- They often have an explicit, contract-defined interpretation in the L1 OSP prover.

For this experiment we introduced a **deterministic, controlled deviation** in Host I/O behavior on the fault-injected node. The honest node retained the standard behavior.

### Determinism and consistency across execution backends

Nitro can execute and validate through multiple backends (e.g., execution engine, JIT, arbitrator). A key engineering requirement for a meaningful dispute is:

> The fault-injected node must be **self-consistent** across all of its own execution backends.

If different backends disagree locally, the node may fail fast or never progress far enough to participate in the dispute protocol.

Therefore, our harness applies the same deterministic “fault” semantics across the relevant execution/validation components.

## Engineering challenges we had to solve (high-level)

### 1) Keeping the node operable while “wrong”

Many systems contain internal sanity checks that assume honest execution. When intentionally testing adversarial behavior, some of these checks must be handled carefully so they don’t prematurely stop the node before the protocol reaches OSP.

In our harness, we treated the fault injection as a dedicated test mode and ensured the system could remain operable long enough to complete the dispute lifecycle.

### 2) Artifact selection and environment coherence

OSP and intermediate proofs depend on machine artifacts (e.g., replay binaries / machine images). In local development and test networks, it is common for on-chain configuration and local build artifacts to diverge.

Our testing setup ensures the dispute machinery can obtain coherent artifacts to produce the required intermediate commitments and proofs. (Implementation details are omitted.)

### 3) Transaction preflight / simulation failure near OSP

Submitting OSP-related transactions typically involves simulation (e.g., `eth_estimateGas`) as a safety check. When the fault-injected node constructs a proof that does not match L1 semantics, simulation can revert—yet we still want to submit the transaction so L1 can adjudicate.

Our harness accounts for this operational reality and ensures the experiment can proceed to on-chain adjudication. (Implementation details are omitted.)

## Results and observations

### What we verified

- A deterministic Host I/O divergence can trigger an execution disagreement that proceeds through BOLD’s narrowing phases.
- The dispute can reach OSP, where the on-chain prover semantics remain authoritative.
- Engineering details (backend consistency, artifact coherence, and transaction submission behavior) materially affect whether an experiment reaches OSP in practice.

### Why this matters

This provides confidence that:

- The dispute protocol is not only theoretically sound, but also **operationally reachable** under fault conditions.
- L1 adjudication provides a clear, deterministic final answer even when off-chain nodes behave incorrectly.

## Safety and disclosure note

This is a **public summary**. We intentionally do **not** publish:

- The exact Host I/O operation chosen,
- The exact mutation strategy or parameters,
- Configuration toggles / operational steps to reproduce the fault,
- Any guidance that could enable misuse on live networks.

If you are a partner/auditor and need more details for a legitimate review, please contact the team through an appropriate disclosure channel.

## Next steps (potential improvements)

- Expand the fault model beyond Host I/O (e.g., cross-module interactions, metering, edge cases).
- Automate multi-backend consistency checks as part of CI for regression detection.
- Improve observability for challenge progression (especially around stage transitions) to shorten time-to-debug.

