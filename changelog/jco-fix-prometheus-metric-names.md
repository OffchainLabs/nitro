### Fixed
- Fix invalid Prometheus metric names in filtering-report and transaction-filterer components; hyphens in geth metric paths survive the `/`→`_` translation and produce names that violate the Prometheus spec
- Add `prometheusmetrics` linter to catch invalid metric names at compile time
