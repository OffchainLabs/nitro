### Added
- Add per-worker slowdown on consecutive retryable HTTP errors from the external endpoint to avoid hammering a degraded service and consuming max receive count quota unnecessarily.
- Send non-retryable HTTP errors to an optional poison queue for manual inspection or reprocessing.
