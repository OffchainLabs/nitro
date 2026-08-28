### Fixed
- Fixed a race where a concurrent head update (such as a message digest or a reorg) could move the chain head while the sequencer was reading chain state to build a block. On sequencer nodes, all head-moving execution calls now hold the block-creation lock, and each block-creation turn reads the header and state as one consistent snapshot.
