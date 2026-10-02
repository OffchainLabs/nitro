### Fixed
- Feed server no longer lets a client silently miss messages: a confirmed sequence number message received right after the backlog no longer ends the client's backlog catch-up, and a client that could not be sent one message of a multi-message broadcast is now disconnected even if the later messages could be sent to it.
