# ExecutionChallenge

The `ExecutionChallenge` is an instance of a `ChallengeCore` (see `ChallengeCore.md` for details).
It's instantiated from a BlockChallenge with a start and end machine hash.
Its challenge unit is a machine, which is hashed in `Machines.sol`.  Its step is the execution of
a single WAVM instruction.

Once the ExecutionChallenge has been bisected down to an individual step,
`oneStepProveExecution` can be called by the current responder.
The current responder must provide proof data to execute a step of the machine.
If executing that step ends in a different state than was previously asserted,
the current responder wins the challenge.

Note that for the time being, winning the challenge isn't instant.
Instead, it simply makes the current responder the winner's opponent,
and sets the state hash to 0. In that state the party does not have any
value moves, so it will eventually lose by timeout.
This is done as a precaution, so that if a challenge is resolved incorrectly,
there is time to diagnose and fix the error with a contract upgrade.
