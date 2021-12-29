# BlockChallenge

The `BlockChallenge` is an instance of a `ChallengeCore` (see `ChallengeCore.md` for details).
It begins with a start global state, a start machine status, an end global state, and an end machine status.
It stores the start global state and end global state in storage so they can be accessed later.
Its challenge unit is a "block state", which consists of a global state and a machine status.
The block state hash function can be found in `ChallengeLib` `blockStateHash`.

Once the BlockChallenge has been bisected down to an individual step,
`challengeExecution` can be called by the current responder.
This operates similarly to a bisection in that the responder must provide a competing global state and machine state,
but it uses that information to create an `ExecutionChallenge` and transfer control to it.
From that point on, the BlockChallenge is inacessible, and any operations must go through the ExecutionChallenge.
See `ExecutionChallenge.md` for details.
