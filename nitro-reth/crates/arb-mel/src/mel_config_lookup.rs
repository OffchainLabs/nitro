//! Parsing `MELConfigSet` logs from a parent-chain block.

use alloy_consensus::Header;
use alloy_primitives::Address;
use alloy_sol_types::{SolEvent, sol};

use crate::{LogsFetcher, MelError, MelResult};

sol! {
    #[allow(missing_docs)]
    #[derive(Debug)]
    event MELConfigSet(
        uint16 indexed melVersion,
        address indexed inbox,
        address indexed sequencerInbox,
        uint64 activationBlock
    );
}

/// A MEL configuration carried by the rollup admin contract's `MELConfigSet`
/// event, activating or upgrading MEL consensus at a given parent-chain block.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct MelConfig {
    pub mel_version: u16,
    pub inbox: Address,
    pub sequencer_inbox: Address,
    pub activation_block: u64,
}

/// Scans the logs of the given parent-chain block for a `MELConfigSet` event
/// and returns the config it carries, or `None` if the block emits no such
/// event. The log fetcher is expected to have already filtered logs to the
/// rollup address, so only the event topic is matched here.
pub(crate) fn parse_mel_config_from_block<L>(
    parent_chain_header: &Header,
    logs_fetcher: &L,
) -> MelResult<Option<MelConfig>>
where
    L: LogsFetcher,
{
    let logs = logs_fetcher.logs_for_block_hash(parent_chain_header.hash_slow())?;
    for log in &logs {
        if log.topics().first() != Some(&MELConfigSet::SIGNATURE_HASH) {
            continue;
        }
        let event = MELConfigSet::decode_log(&log.inner)
            .map_err(|source| MelError::AbiDecode {
                event: "MELConfigSet",
                source,
            })?
            .data;
        return Ok(Some(MelConfig {
            mel_version: event.melVersion,
            inbox: event.inbox,
            sequencer_inbox: event.sequencerInbox,
            activation_block: event.activationBlock,
        }));
    }
    Ok(None)
}

#[cfg(test)]
mod tests {
    use alloy_primitives::{B256, LogData};
    use alloy_rpc_types_eth::Log;

    use super::*;
    use crate::test_utils::{MockLogs, rpc_log};

    /// Parent-chain address the tests emit their config events from.
    fn rollup() -> Address {
        Address::repeat_byte(0xCD)
    }

    fn mel_config_log(version: u16, inbox: Address, seq_inbox: Address, activation: u64) -> Log {
        let ev = MELConfigSet {
            melVersion: version,
            inbox,
            sequencerInbox: seq_inbox,
            activationBlock: activation,
        };
        rpc_log(rollup(), ev.encode_log_data())
    }

    #[test]
    fn parses_mel_config_event() -> MelResult<()> {
        let inbox = Address::repeat_byte(0x11);
        let seq_inbox = Address::repeat_byte(0x22);
        let logs = MockLogs {
            block_logs: vec![mel_config_log(3, inbox, seq_inbox, 42)],
            ..Default::default()
        };
        let cfg = parse_mel_config_from_block(&Header::default(), &logs)?;
        assert_eq!(
            cfg,
            Some(MelConfig {
                mel_version: 3,
                inbox,
                sequencer_inbox: seq_inbox,
                activation_block: 42,
            })
        );
        Ok(())
    }

    #[test]
    fn returns_none_when_no_config_event() -> MelResult<()> {
        // A log with an unrelated topic at the rollup address is ignored.
        let unrelated = rpc_log(
            rollup(),
            LogData::new_unchecked(vec![B256::repeat_byte(0xEE)], Default::default()),
        );
        let logs = MockLogs {
            block_logs: vec![unrelated],
            ..Default::default()
        };
        assert_eq!(parse_mel_config_from_block(&Header::default(), &logs)?, None);
        Ok(())
    }

    #[test]
    fn returns_first_config_event() -> MelResult<()> {
        let logs = MockLogs {
            block_logs: vec![
                mel_config_log(1, Address::repeat_byte(0x11), Address::repeat_byte(0x22), 7),
                mel_config_log(2, Address::repeat_byte(0x33), Address::repeat_byte(0x44), 8),
            ],
            ..Default::default()
        };
        let cfg = parse_mel_config_from_block(&Header::default(), &logs)?.unwrap();
        assert_eq!(cfg.mel_version, 1);
        assert_eq!(cfg.activation_block, 7);
        Ok(())
    }
}
