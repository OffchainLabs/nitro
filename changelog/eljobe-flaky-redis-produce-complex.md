### Ignored
- Fix flaky TestRedisProduceComplex by treating a lost SetResult/SetError race (redis key already set) as the expected at-least-once outcome instead of a test failure
