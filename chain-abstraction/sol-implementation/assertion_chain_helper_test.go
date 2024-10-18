package solimpl

import protocol "github.com/offchainlabs/bold/chain-abstraction"

func (a *AssertionChain) SetBackend(b protocol.ChainBackend) {
	a.backend = b
}
