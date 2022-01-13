# ChallengeCore

_**Note:** the term bisection in this document is used for clarity but refers to a dissection of any degree._

`ChallengeCore` is an abstract contract which contains the basis for bisecting down to a single point in a challenge.
A generic challenge must be over something which advances from one hash to another. A hash of the state at
a single point in time is refered to as a segment.
In practice, a challenge is either a block challenge (where the unit step is creating and 
processing one block),
or an execution challenge (where the unit step is executing one WAVM instruction).

The `ChallengeLib` helper library contains a `hashChallengeState` method which hashes a list of segment hashes,
a start position, and a total segments length.
This is enough information to infer the position of each segment hash.
The challenge "degree" refers to the number of segment hashes minus one.
The distance (in steps) between one segment and the next is `floor(segmentsLength / degree)`, except for the
last pair of segments, where `segmentsLength % degree` is added to the normal distance, so that
the total distance is `segmentsLength`.

A challenge begins with only two segments (a degree of one), which is the asserter's initial assertion.
Then, the bisection game begins on the challenger's turn.
In each round of the game, the current responder must choose an adjacent pair of segments to challenge.
By doing so, they are disputing their opponent's claim that starting with the first segment and executing
for the specified distance (number of steps) will result in the second segment. At this point the two parties
agree on the correctness of the first segment but disagree about the correctness of the second segment.
The responder must provide a bisection with a start segment equal to the first segment, but an end segment
different from the second segment.
In doing so, they break the challenge down into smaller distances, and it becomes their opponent's turn.
Each bisection must have degree `min(40, numStepsInChallengedSegment)`, ensuring the challenge makes progress.

In addition, a segment with a length of only one step cannot be bisected.
That case is challenge type specific, as it depends on the nature of a step.

Note that unlike in a traditional bisection protocol, where one party proposes segments and the other decides which to challenge,
this protocol is symmetric in that both players take turns deciding where to challenge and proposing bisections 
when challenging.
