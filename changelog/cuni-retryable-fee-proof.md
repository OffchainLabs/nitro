### Added

- `cuni-proofs/retryable-fee/`: retryable-ticket submission pricing stated as a
  single CuNi law (`retryable_fee.cuni`) — submission fee, submission refund,
  L2 gas cost, ArbOS v11+ infra-fee split, and gas-price refund — proven
  byte-identical against the implementation on a 34-vector battery (plus 16
  refusal cases) across Python, JavaScript, and Go seats, with a Go test
  running the `submission_fee` vectors against `retryables.RetryableSubmissionFee`.
