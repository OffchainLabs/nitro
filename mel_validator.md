# MEL Block Validator Notes

1. Block validator needs to find out if its caught up
2. It asks the tracker for batch count and streamer for processed message count
3. If caught up, launches threads
4. First thread:
   1. Creates validation entries with Status = CREATED
5. Second thread:
   1. Prepares to record preimages from the latest send message index up to some NUM_RECORD_BLOCKS value
   2. Then, it tells the validation entry to do the recording using the prepared recorder
6. Third thread:
   1. Sends the validations by instantiating spawners and sending requests to them
7. Fourth thread:
   1. Check the results of validations that have been completed, and set the latest valid gs
8. In MEL recording:
   1. We can choose what to record

## Design Ideas

- Keep block validator mostly the same
- Add a new MEL validator that specifically validates MEL execution
- Block validator will ask MEL validator to see if it is caught up on startup
- Create validation entry will ask transaction streamer and the MEL validator to see if it can create an entry
- If MEL validation is enabled, the block validator will depend on a MEL validator

MEL Validator
- 4 threads similar to the block validator
- Each thread can be an FSM
- We need a MEL recorder implementation

func (crt *CreateValidationsFSM) Start() {
	CallIteratively(func() {
		nextActionInterval, err := crt.Act()
		if err != nil {
			return time.Second
		}
		select {
		case <-manualTrigger:
			return 0 // Act again immediately.
		case <-time.After(nextActionInterval):
			return nextActionInterval
		}
	})
}

## Caveats

- Dangerous flags for assuming init valid
- BoLD can tell us that we should update our latest valid global state, and this should retrigger all the threads

## Global State Pre vs. Post MEL

Block validator GS:

GlobalState {
    BlockHash // Result of executing message at batch N, pos M.
    SendRoot // Used for the bridge (can ignore)
    Batch // The batch index.
    PosInBatch // Message index within a batch (not global message index)
	MELStateRoot // EMPTY.
}

MEL GS:

GlobalState {
    BlockHash // EMPTY.
    SendRoot // EMPTY.
    Batch // MAX_UINT64: unused.
    PosInBatch // Global message index N.
	MELStateRoot // MELStateRoot that read message index N.
}

Staker:

- Needs to submit assertions to L1 that claim validity of MEL + block execution
- They need to unify the mel and block validator global states.

Assertion {
	BeforeState: GlobalState
	AfterState: GlobalState
	ParentChainBlockHash: common.Hash
}