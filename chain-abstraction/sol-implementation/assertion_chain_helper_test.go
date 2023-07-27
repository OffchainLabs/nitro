package solimpl

func (a *AssertionChain) SetBackend(b ChainBackend) {
	a.backend = b
}
