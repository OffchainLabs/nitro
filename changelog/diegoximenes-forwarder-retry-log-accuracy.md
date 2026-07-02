### Changed
- The transaction forwarder (`TxForwarder`) now logs whether it will actually try another target: forwarding errors that are returned to the client without retrying (e.g. filtered tx, out of gas, nonce errors) are logged distinctly from transport-level errors that trigger a retry against the next target, instead of always logging "trying different target".
