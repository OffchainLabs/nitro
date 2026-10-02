### Fixed
- Fix Prometheus scrape failures caused by `iostat` occasionally parsing a numeric string as a device name; `parseStream` now skips rows whose device name parses as a float
- Sanitize `iostat` device names via the shared `metricsutil.CanonicalizeMetricName` helper in `RegisterAndPopulateMetrics` instead of only replacing hyphens, so any invalid Prometheus metric-name character is handled
