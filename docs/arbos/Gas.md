# Gas
TODO

## Gas Estimating Retryables



### NodeInterface.sol
To avoid creating new RPC methods for client-side tooling, nitro geth's [`InterceptRPCMessage`][InterceptRPCMessage_link] hook provides an opportunity to swap out the message its handling before deriving a transaction from it. ArbOS uses this hook to detect messages sent to the address `0xc8`, the location of a fictional contract ... TODO

[InterceptRPCMessage_link]: todo
