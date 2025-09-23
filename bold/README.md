# BOLD

This package implements Offchain Labs' BOLD (Bounded Liquidity Delay) Protocol: a dispute system to enable permissionless validation of Arbitrum chains. It is an efficient, all-vs-all challenge protocol that enables anyone on Ethereum to challenge invalid rollup state transitions.

BOLD provides a fixed, upper-bound on challenge confirmations for Arbitrum chains.

Given state transitions are deterministic, this guarantees only one correct result for any given assertion. An **honest participant** will always win against malicious entities when challenging assertions posted to the settlement chain. 

## Directory Structure

For our research specification of BOLD, see [BOLDChallengeProtocol.pdf](docs/research-specs/BOLDChallengeProtocol.pdf).

For our technical deep dive into BOLD, see [TechnicalDeepDive.pdf](docs/research-specs/TechnicalDeepDive.pdf)

For documentation on the economics of BOLD, see [Economics.pdf](docs/research-specs/Economics.pdf)

For detailed information on how our code is architected, see [ARCHITECTURE.md](docs/ARCHITECTURE.md).

```
api/ 
    API for monitoring and visualizing challenges
assertions/
    Logic for scanning and posting assertions
chain-abstraction/
    High-level wrappers around Solidity bindings for the Rollup contracts
challenge-manager/
    All logic related to challenging, managing challenges
containers/
    Data structures used in the repository, including FSMs
contracts/
    All Rollup / challenge smart contracts
docs/
    Diagrams and architecture
layer2-state-provider/
    Interface to request state and proofs from an L2 backend
math/
    Utilities for challenge calculations
runtime/
    Tools for managing function lifecycles
state-commitments/
    Proofs, history commitments, and Merkleizations
testing/
    All non-production code
third_party/
    Build artifacts for dependencies
time/
    Abstract time utilities
```

## Research Specification

BOLD has an accompanying research specification that outlines the foundations of the protocol in more detail, found under [docs/research-specs/BOLDChallengeProtocol.pdf](./docs/research-specs/BOLDChallengeProtocol.pdf).


## Security Audit

BOLD has been audited by [Trail of Bits](https://www.trailofbits.com/) as of commit [60f97068c12cca73c45117a05ba1922f949fd6ae](https://github.com/OffchainLabs/bold/commit/60f97068c12cca73c45117a05ba1922f949fd6ae), and a more updated audit is being completed, to be finalized in the coming few weeks.

The audit report can be found under [docs/audits/TrailOfBitsAudit](./docs/audits/TrailOfBitsAudit.pdf).

## Credits

Huge credits on this project go to those who created BOLD and were involved in its implementation: Ed Felten, Yafah Edelman, Chris Buckland, Harry Ng, Lee Bousfield, Terence Tsao, Mario Alvarez, Preston Van Loon, Mahimna Kelkar, Aman Sanghi, Daniel Goldman, Raul Jordan, Henry Arneson, Derek Lee, Victor Shoup
