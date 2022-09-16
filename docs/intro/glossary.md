
# Glossary of Arbitrum Terms

- **Arbitrum Chain**: A Layer 2 EVM environment running on Ethereum using Arbitrum technologies. Arbitrum Chains come in two forms, rollup and anytrust, depending on a user's needs.

- **Arbitrum AnyTrust**: Arbitrum protocol in which data availability is managed by a permissioned set of parties; compared to Arbitrum Rollup, not strictly trustless, but offers lower fees (e.g., Arbitrum Nova). 

- **Arbitrum Rollup**: Trustless Arbitrum L2 protocol in which participation is permissionless and underlying layer is used for data availability (e.g., Arbitrum One).

- **Arbitrum One**: The first Arbitrum Rollup chain running on Ethereum mainnet! (Currently in Beta).

- **Arbitrum Nova**: The first Arbitrum AnyTrust chain running on Ethereum mainnet! (Currently in Beta).

- **Arbitrum Full Node**: A party who keeps track of the state of an Arbitrum chain and receives remote procedure calls (RPCs) from clients. Analogous to a non-staking L1 Ethereum node.

- **ArbOS**: Layer 2 "operating system" that trustlessly handles system-level operations; includes the ability to emulate the EVM.

- **Arbitrum Classic**: [Old Arbitrum stack](https://github.com/OffchainLabs/arbitrum) that used custom virtual machine ("AVM"); no public Arbitrum chain uses the classic stack as of 8/31/2022 (they instead use Nitro).

- **Nitro**: Current Arbitrum tech stack; runs a fork of Geth directly on L2 and uses WebAssembly as its underlying VM for fraud proofs.

- **Data Availability Committee (DAC)**: Permissioned set of parties responsibly for data availability in an AnyTrust chain.

- **Data Availability Certificate**: Signed promise from a DAC of relevant data's availability as of a given Inbox hash. 

- **Chain state**: A particular point in the history of an Arbitrum Chain. A chain state corresponds to a sequence of assertions that have been made, and a verdict about whether each of those assertions was accepted.

- **Client**: A program running on a user's machine, often in the user's browser, that interacts with contracts on an Arbitrum chain and provides a user interface.

- **Rollup Protocol**: Protocol for tracking the tree of assertions in an Arbitrum chain and their confirmation status.

- **Speed Limit**: Target L2 computation limit for an Arbitrum chain. When computation exceeds this limit, fees rise, ala [EIP-1559](https://notes.ethereum.org/@vbuterin/eip-1559-faq).

## Proving Fraud

- **RBlock**: An assertion by an Arbitrum validator that represents a claim about an Arbitrum chain's state.

- **L2 Block**: Data structure that represents a group of L2 transactions (analogous to L1 blocks).

- **Challenge**: When two stakers disagree about the correct verdict on an assertion, those stakers can be put in a challenge. The challenge is refereed by the contracts on L1. Eventually one staker wins the challenge. The loser forfeits their stake. Half of the loser's stake is given to the winner, and the other half is burned.

- **Confirmation**: The final decision by an Arbitrum chain to accept an RBlock as being a settled part of the chain's history. Once an assertion is confirmed, any L2 to L1 messages (i.e., withdrawals) can be executed.

- **Challenge Period**: Window of time (1 week on Arbitrum One) over which an asserted RBlock can be challenged, and after which the RBlock can be confirmed.

- **Dissection**: Process by which two challenging parties interactively narrow down their disagreement to a single computational step.

- **One Step Proof**: Final step in a challenge; a single operation of the L2 VM (Wasm) is executed on L1, and the validity of its state transition is verified.


- **Staker**: A party who deposits a stake, in ETH, to vouch for a particular RBlock in an Arbitrum Chain. A party who stakes on a false RBlock can expect to lose their stake. An honest staker can recover their stake once the node they are staked on has been confirmed.


- **Active Validator**: A party who makes staked, disputable assertions about the state of the Arbitrum chain; i.e., proposing state updates or challenging the validity of assertions. (Not to be confused with the Sequencer)

- **Defensive Validator**: A validator that watches the Arbitrum chain and takes action (i.e., stake and challenges) only when and if an invalid assertion occurs.

- **Watchtower Validator**: A validator that never stakes / never takes on chain action, who raises the alarm (by whatever off-chain means it chooses) if it witnesses an invalid assertion.

## Cross Chain Communication

- **Address Alias**: A deterministically generated address to be used on L2 that corresponds to an address on L1 for the purpose of L1 to L2 cross-chain messaging.

- **Fast Exit / Liquidity Exit**: A means by which a user can bypass Arbitrum's challenge period when withdrawing fungible assets (or more generally, executing some "fungible" L2 to L1 operation); a liquidity provider facilitates an atomic swap of the asset on L2 directly to L1.

- **Outbox**: An L1 contract responsible for tracking outgoing (Arbitrum to Ethereum) messages, including withdrawals, which can be executed by users once they are confirmed. The outbox stores a Merkle Root of all outgoing messages.

- **Retryable Ticket**: An L1 to L2 cross chain message initiated by an L1 transaction sent to an Arbitrum chain for execution (e.g., a token deposit).

- **Retryable Autoredeem**: The "automatic" execution of a retryable ticket on Arbitrum (using provided ArbGas).

## Token Bridging

- **Arb Token Bridge**: A series of contracts on Ethereum and Arbitrum for trustlessly moving tokens between the L1 and L2.

- **Token Gateway**: A pair of contracts in the token bridge — one on L1, one on L2 — that provide a particular mechanism for handling the transfer of tokens between layers. Token gateways currently active in the bridge are the StandardERC20 Gateway, the CustomERC20 Gateway, and the WETH Gateway.

- **Gateway Router**: Contracts in the token bridge responsible for mapping tokens to their appropriate gateways.

- **Standard Arb-Token**: An L2 token contract deployed via the StandardERC20 gateway; offers basic ERC20 functionality in addition to deposit / withdrawal affordances.

- **Custom Arb-Token**: Any L2 token contract registered to the Arb Token Bridge that isn't a standard arb-token (i.e., a token that uses any gateway other than the StandardERC20 Gateway).

## Transaction Ordering

- **Batch**: A group of L2 transactions posted in a single L1 transaction by the Sequencer.

- **Fair Ordering Algorithm**: BFT algorithm in which a committee comes to consensus on transaction ordering; current single-party Sequencer on Arbitrum one will eventually be replaced by a fair-ordering committee.

- **Forced-Inclusion**: Censorship resistant path for including a message into L2; bypasses any Sequencer involvement.

- **Sequencer**: An entity (currently a single-party on Arbitrum One) given rights to reorder transactions in the Inbox over a small window of time, who can thus give clients sub-blocktime soft confirmations. (Not to be confused with a validator)

- **Soft Confirmation**: A semi-trusted promise from the Sequencer to post a user's transaction in the near future; soft-confirmations happen prior to posting on L1, and thus can be given near-instantaneously (i.e., faster than L1 block times)

- **Slow Inbox**: Sequence of L1 initiated message that offer an alternative path for inclusion into the fast Inbox.

- **Fast Inbox**: Contract that holds a sequence of messages sent by clients to the contracts on an Arbitrum Chain; message can be put into the Inbox directly by the Sequencer or indirectly through the slow inbox.
