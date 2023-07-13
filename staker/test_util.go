package staker

func NewStatelessBlockValidatorStruct(
	inbox InboxTrackerInterface,
	streamer TransactionStreamerInterface,
) *StatelessBlockValidator {
	return &StatelessBlockValidator{
		inboxTracker: inbox,
		streamer:     streamer,
	}
}
