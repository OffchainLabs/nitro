use std::collections::HashMap;

use alloy_primitives::B256;

use super::{DaError, DaReader, Payload, Preimages, Result};

/// Builds a fresh [`DaError`] each time a registered failure is read.
///
/// A closure rather than a stored `DaError` because `DaError` isn't `Clone`
/// (it holds boxed trait objects).
type ErrorFactory = Box<dyn Fn() -> DaError + Send + Sync>;

/// An in-memory [`DaReader`] for tests.
///
/// Batches are registered up front with the `with_*` builders, then read back
/// through the trait. Each batch is keyed by the full
/// `(batch_num, batch_block_hash, sequencer_msg)` triple the trait looks it up
/// by, so lookups can happen in any order.
///
/// A batch can be registered as a success (via [`Self::with_batch`]) or as a
/// failure (via [`Self::with_error`] for an arbitrary [`DaError`], or
/// [`Self::with_provider_error`] for the common "fail with this message" case).
/// A batch that was never registered reads back as an empty payload and empty
/// preimages, mirroring the "missing field is empty" behaviour of the RPC
/// reader.
#[derive(Default)]
pub struct MockDaReader {
    batches: HashMap<(u64, B256, Vec<u8>), std::result::Result<(Payload, Preimages), ErrorFactory>>,
}

impl MockDaReader {
    /// Creates an empty reader.
    pub fn new() -> Self {
        Self::default()
    }

    /// Registers the payload and preimages recovered for a batch.
    pub fn with_batch(
        &mut self,
        batch_num: u64,
        batch_block_hash: B256,
        sequencer_msg: &[u8],
        payload: Payload,
        preimages: Preimages,
    ) -> &mut Self {
        self.batches.insert(
            (batch_num, batch_block_hash, sequencer_msg.to_vec()),
            Ok((payload, preimages)),
        );
        self
    }

    /// Registers a batch that fails to recover, minting the error from `error`
    /// on each read. Use this to inject any [`DaError`] variant.
    pub fn with_error(
        &mut self,
        batch_num: u64,
        batch_block_hash: B256,
        sequencer_msg: &[u8],
        error: impl Fn() -> DaError + Send + Sync + 'static,
    ) -> &mut Self {
        self.batches.insert(
            (batch_num, batch_block_hash, sequencer_msg.to_vec()),
            Err(Box::new(error)),
        );
        self
    }

    /// Registers a batch that fails to recover with `message` surfaced as a
    /// provider error — the failure shape a real reader produces. A convenience
    /// wrapper over [`Self::with_error`].
    pub fn with_provider_error(
        &mut self,
        batch_num: u64,
        batch_block_hash: B256,
        sequencer_msg: &[u8],
        message: impl Into<String>,
    ) -> &mut Self {
        let message = message.into();
        self.with_error(batch_num, batch_block_hash, sequencer_msg, move || {
            DaError::provider(std::io::Error::other(message.clone()))
        })
    }

    /// Resolves the outcome registered for a batch, defaulting an unregistered
    /// batch to an empty payload and empty preimages.
    fn outcome(
        &self,
        batch_num: u64,
        batch_block_hash: B256,
        sequencer_msg: &[u8],
    ) -> Result<(Payload, Preimages)> {
        match self
            .batches
            .get(&(batch_num, batch_block_hash, sequencer_msg.to_vec()))
        {
            Some(Ok(data)) => Ok(data.clone()),
            Some(Err(make_error)) => Err(make_error()),
            None => Ok((Payload::new(), Preimages::new())),
        }
    }
}

#[async_trait::async_trait]
impl DaReader for MockDaReader {
    async fn recover_payload(
        &self,
        batch_num: u64,
        batch_block_hash: B256,
        sequencer_msg: &[u8],
    ) -> Result<Payload> {
        self.outcome(batch_num, batch_block_hash, sequencer_msg)
            .map(|(payload, _)| payload)
    }

    async fn collect_preimages(
        &self,
        batch_num: u64,
        batch_block_hash: B256,
        sequencer_msg: &[u8],
    ) -> Result<Preimages> {
        self.outcome(batch_num, batch_block_hash, sequencer_msg)
            .map(|(_, preimages)| preimages)
    }

    async fn recover_payload_and_preimages(
        &self,
        batch_num: u64,
        batch_block_hash: B256,
        sequencer_msg: &[u8],
    ) -> Result<(Payload, Preimages)> {
        self.outcome(batch_num, batch_block_hash, sequencer_msg)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::PreimageType;

    /// The batch coordinates shared across the tests below.
    const BATCH_NUM: u64 = 7;
    fn block_hash() -> B256 {
        B256::repeat_byte(0x11)
    }
    const SEQ_MSG: &[u8] = &[0xde, 0xad, 0xbe, 0xef];

    /// Builds a single-entry preimages map for assertions.
    fn preimages() -> Preimages {
        let hash = B256::repeat_byte(0x22);
        Preimages::from([(
            PreimageType::Keccak256,
            HashMap::from([(hash, vec![1, 2, 3])]),
        )])
    }

    #[tokio::test]
    async fn recover_payload_returns_registered_payload() {
        let mut mock = MockDaReader::new();
        mock.with_batch(
            BATCH_NUM,
            block_hash(),
            SEQ_MSG,
            vec![1, 2, 3],
            Preimages::new(),
        );

        assert_eq!(
            mock.recover_payload(BATCH_NUM, block_hash(), SEQ_MSG)
                .await
                .unwrap(),
            vec![1, 2, 3]
        );
    }

    #[tokio::test]
    async fn collect_preimages_returns_registered_preimages() {
        let mut mock = MockDaReader::new();
        mock.with_batch(
            BATCH_NUM,
            block_hash(),
            SEQ_MSG,
            Payload::new(),
            preimages(),
        );

        let result = mock
            .collect_preimages(BATCH_NUM, block_hash(), SEQ_MSG)
            .await
            .unwrap();
        assert_eq!(result, preimages());
    }

    #[tokio::test]
    async fn recover_payload_and_preimages_returns_both() {
        let mut mock = MockDaReader::new();
        mock.with_batch(BATCH_NUM, block_hash(), SEQ_MSG, vec![1, 2, 3], preimages());

        let (payload, images) = mock
            .recover_payload_and_preimages(BATCH_NUM, block_hash(), SEQ_MSG)
            .await
            .unwrap();
        assert_eq!(payload, vec![1, 2, 3]);
        assert_eq!(images, preimages());
    }

    #[tokio::test]
    async fn with_provider_error_surfaces_as_provider_error() {
        let mut mock = MockDaReader::new();
        mock.with_provider_error(BATCH_NUM, block_hash(), SEQ_MSG, "batch not found");

        let payload_err = mock
            .recover_payload(BATCH_NUM, block_hash(), SEQ_MSG)
            .await
            .unwrap_err();
        assert!(matches!(payload_err, DaError::Provider(_)));
        // The registered message is carried through.
        assert!(payload_err.to_string().contains("batch not found"));

        let preimages_err = mock
            .collect_preimages(BATCH_NUM, block_hash(), SEQ_MSG)
            .await
            .unwrap_err();
        assert!(matches!(preimages_err, DaError::Provider(_)));

        let both_err = mock
            .recover_payload_and_preimages(BATCH_NUM, block_hash(), SEQ_MSG)
            .await
            .unwrap_err();
        assert!(matches!(both_err, DaError::Provider(_)));
    }

    #[tokio::test]
    async fn with_error_replays_arbitrary_variant() {
        let mut mock = MockDaReader::new();
        // A non-transport variant, to show any DaError can be injected.
        mock.with_error(BATCH_NUM, block_hash(), SEQ_MSG, || {
            DaError::Base64(base64::DecodeError::InvalidPadding)
        });

        // Re-read the same batch across calls and methods; the factory mints a
        // fresh error each time.
        for _ in 0..2 {
            let err = mock
                .recover_payload(BATCH_NUM, block_hash(), SEQ_MSG)
                .await
                .unwrap_err();
            assert!(matches!(err, DaError::Base64(_)));
        }
        assert!(matches!(
            mock.collect_preimages(BATCH_NUM, block_hash(), SEQ_MSG)
                .await
                .unwrap_err(),
            DaError::Base64(_)
        ));
    }

    #[tokio::test]
    async fn unregistered_batch_reads_back_empty() {
        let mock = MockDaReader::new();

        assert!(
            mock.recover_payload(BATCH_NUM, block_hash(), SEQ_MSG)
                .await
                .unwrap()
                .is_empty()
        );
        assert!(
            mock.collect_preimages(BATCH_NUM, block_hash(), SEQ_MSG)
                .await
                .unwrap()
                .is_empty()
        );
        let (payload, images) = mock
            .recover_payload_and_preimages(BATCH_NUM, block_hash(), SEQ_MSG)
            .await
            .unwrap();
        assert!(payload.is_empty());
        assert!(images.is_empty());
    }

    #[tokio::test]
    async fn lookup_uses_the_full_triple() {
        let mut mock = MockDaReader::new();
        mock.with_batch(
            BATCH_NUM,
            block_hash(),
            SEQ_MSG,
            vec![1, 2, 3],
            Preimages::new(),
        );

        // Wrong block hash
        assert!(
            mock.recover_payload(BATCH_NUM, B256::repeat_byte(0xff), SEQ_MSG)
                .await
                .unwrap()
                .is_empty()
        );
        // Wrong sequencer message
        assert!(
            mock.recover_payload(BATCH_NUM, block_hash(), &[0x00])
                .await
                .unwrap()
                .is_empty()
        );
        // Wrong batch number
        assert!(
            mock.recover_payload(BATCH_NUM + 1, block_hash(), SEQ_MSG)
                .await
                .unwrap()
                .is_empty()
        );
    }

    #[tokio::test]
    async fn distinct_keys_do_not_collide() {
        let mut mock = MockDaReader::new();
        mock.with_batch(1, block_hash(), SEQ_MSG, vec![0xaa], Preimages::new());
        mock.with_batch(2, block_hash(), SEQ_MSG, vec![0xbb], Preimages::new());

        assert_eq!(
            mock.recover_payload(1, block_hash(), SEQ_MSG)
                .await
                .unwrap(),
            vec![0xaa]
        );
        assert_eq!(
            mock.recover_payload(2, block_hash(), SEQ_MSG)
                .await
                .unwrap(),
            vec![0xbb]
        );
    }
}
