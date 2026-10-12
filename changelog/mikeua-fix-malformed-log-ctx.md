### Fixed
- Fix log calls that dropped or mislabelled context: a missing `err` value when reading time before flush, bare values in the zeroheavy decoder and refund-undo errors, and BoLD challenge-confirmation logs that passed `fields` without `...` and accumulated stale `err` entries across retries
