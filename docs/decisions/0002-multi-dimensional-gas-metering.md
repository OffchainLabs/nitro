# Multi-Dimensional (MultiGas) Gas Metering with Resource Kinds

## Context and Problem Statement

As part of the implementation of a constraint-based gas pricing model, Ethereum’s traditional single-dimensional gas accounting (`uint64`) is being replaced by a multi-dimensional system in which gas is tracked separately for distinct resource categories. The objective is to isolate and measure consumption across orthogonal resource axes, such as computation, state access, state growth, and historical data appendage. This separation enables fine-grained pricing adjustments per resource in response to network load or policy considerations.

## Decision Outcome

A multi-dimensional gas metering approach is adopted, introducing distinct `ResourceKind` categories. Each opcode’s dynamic gas cost is mapped to one or more resource kinds. The following resource kinds have been identified:

- ResourceKindComputation. Represents pure computational effort, CPU-bound operations that do not mutate global state: 
    - Opcode execution
    - Memory expansion
    - Call gas forwarding (EIP-150)
    - Value transfers (unless to empty accounts, then it's StorageGrowth)
    - Contract init code execution (CREATE, CREATE2)
    - Hashing
    - Bloom filter updates

- ResourceKindStorageAccess. Represents read access to the global state:
    - Account lookups (CALL, EXTCODESIZE, BALANCE)
    - Storage slot reads
    - Storage slot writes (nonzero → nonzero and nonzero → zero)
    - Witness generation for reads (e.g. Verkle/stateless mode)
    - Access list updates (EIP-2929/2930)
    - Verkle proof traversal
    - Target address resolution (DELEGATECALL, STATICCALL)

- ResourceKindStorageGrowth. Includes operations that increase the persistent state size:
    - New account creation
    - Storage slot writes (zero → nonzero)
    - Merkle/Verkle trie growth (EIP-4762)
    - Contract deployment deposit cost

- ResourceKindHistoryGrowth. Represents writes to the append-only event log history:
    - Event logs (LOG0–LOG4)
