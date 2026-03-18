### Fixed
 - Fix ValidationSpawnerRetryWrapper lifecycle: reuse one wrapper per module root instead of creating and leaking one per validation
 - Fix BroadcastClients launching coordination goroutine on child Router's StopWaiter instead of its own
 - Fix ValidationServer and ExecutionSpawner missing StopAndWait for their children
