### Fixed
- Fixed a "send on closed channel" panic in the BOLD event `Producer`: `Next` no longer closes the subscription channel that in-flight `Broadcast` goroutines send to.
- The challenge manager now runs the block-notifier's `Start` loop, so finished subscriptions are reaped rather than accumulating for the life of the process.
