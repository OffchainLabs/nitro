# ChallengeCore

_**Note:** the term bisection in this document is used for clarity but refers to a dissection of any degree._

`ChallengeCore` is an abstract contract which contains the basis for bisecting down to a single point in a challenge.
A generic challenge must be over something which advances from one hash to another (refered to as segments).
In practice, this is either a block challenge (where advancing means creating and processing the next block),
or an execution challenge (where advancing means executing the next instruction).

The `ChallengeLib` helper library contains a `hashChallengeState` method which hashes a list of segment hashes,
a start position, and a total segments length.
This is enough information to infer the position of each segment hash.
The challenge "degree" refers to the number of segment hashes minus one.
The distance between one segment and the next is `floor(segmentsLength / degree)`, except for the
last pair of segments, where `segmentsLength % degree` is added to the normal distance, so that
the total distance is `segmentsLength`.

A challenge begins with only two segments (a degree of one), which is the asserter's assertion.
Then, the bisection game begins on the challenger's turn.
In each round of the game, the current responder must chose an adjacent pair of segments to challenge.
By doing so, they are disputing that progressing from the first segment will result in the second segment.
They must provide a bisection with a start segment equal to the original first segment, but an end segment
different from the original second segment.
In doing so, they break the challenge down into smaller units, and it becomes their opponent's turn.
Each bisection must have degree `min(40, numStepsInChallengedSegment)`, ensuring the challenge makes progress.
In addition, a segment with a length of only one step cannot be bisected.
That step is challenge type specific, as it depends on the definition of advancing.

Note that unlike in a traditional bisection protocol, where one part proposes segments and the other decides which to challenge,
this protocol is symmetric in that both players take turn deciding where to challenge and proposing segments at the same time.
