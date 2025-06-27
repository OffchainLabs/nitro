# Old Submitted Espresso Tx RLP Artifact

This directory contains an RLP-encoded data artifact for the `OldSubmittedEspressoTx` struct for backward compatibility testing.

The old struct format can be found at [this commit](https://github.com/EspressoSystems/nitro-espresso-integration/blob/b8c11f3cdb91893f8c1109872f7c46eb6f82e57d/arbutil/espresso_utils.go#L19).

The file `old_submitted_espresso_tx.rlp` was generated using the Go code in file `generate_rlp.go`.

You can regenerate the file by running:

```sh
go run arbutil/testdata/generate_rlp.go
```
