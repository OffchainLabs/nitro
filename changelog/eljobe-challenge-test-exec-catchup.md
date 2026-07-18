### Fixed
- Fixed flaky legacy challenge system tests (`TestMockChallengeManagerAsserter*`, `TestChallengeManagerFull*`) by waiting for both execution nodes to digest all injected batches before creating the challenge
