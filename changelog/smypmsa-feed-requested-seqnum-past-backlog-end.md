### Fixed
- The feed broadcaster no longer sends the entire backlog to a client that requests a sequence number after the end of the backlog. Such a client is already up to date, so it is now sent none of the backlog and only the messages that follow.
