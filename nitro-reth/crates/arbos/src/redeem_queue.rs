//! Per-block queue of scheduled retry transactions.

use std::collections::VecDeque;

/// FIFO queue of the retry txs scheduled by redeems. A scheduled retry runs right after the tx
/// that scheduled it, before the next message tx; retries scheduled by a retry join the back of
/// the queue. The driving loop must drain the queue fully before pulling the next message tx.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct RedeemQueue<T> {
    queue: VecDeque<T>,
}

impl<T> RedeemQueue<T> {
    pub fn new() -> Self {
        Self {
            queue: VecDeque::new(),
        }
    }

    pub fn is_empty(&self) -> bool {
        self.queue.is_empty()
    }

    pub fn len(&self) -> usize {
        self.queue.len()
    }

    /// Appends the retries scheduled by the tx that just executed.
    pub fn schedule(&mut self, txs: impl IntoIterator<Item = T>) {
        self.queue.extend(txs);
    }

    pub fn clear(&mut self) {
        self.queue.clear();
    }

    /// Iterates the pending retries in FIFO order without consuming them.
    pub fn iter(&self) -> impl Iterator<Item = &T> {
        self.queue.iter()
    }

    /// Pops the next scheduled retry whose ticket is still live; dead tickets (already redeemed,
    /// deleted or expired) are dropped. The caller answers the liveness question, keeping this type
    /// state-free.
    pub fn pop_live(&mut self, mut is_live: impl FnMut(&T) -> bool) -> Option<T> {
        while let Some(tx) = self.queue.pop_front() {
            if is_live(&tx) {
                return Some(tx);
            }
        }
        None
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn fresh_queue_is_empty() {
        let queue = RedeemQueue::<u32>::new();
        assert!(queue.is_empty());
        assert_eq!(queue.len(), 0);
    }

    #[test]
    fn pops_in_fifo_order() {
        let mut queue = RedeemQueue::new();
        queue.schedule(["a", "b", "c"]);
        assert_eq!(queue.len(), 3);

        assert_eq!(queue.pop_live(|_| true), Some("a"));
        assert_eq!(queue.pop_live(|_| true), Some("b"));
        assert_eq!(queue.pop_live(|_| true), Some("c"));
        assert_eq!(queue.pop_live(|_| true), None);
        assert!(queue.is_empty());
    }

    #[test]
    fn retries_scheduled_by_a_retry_join_the_back() {
        // Go appends ScheduledTxes after each execution: with [a, b] pending, a retry scheduled
        // while executing "a" runs after "b".
        let mut queue = RedeemQueue::new();
        queue.schedule(["a", "b"]);

        assert_eq!(queue.pop_live(|_| true), Some("a"));
        queue.schedule(["scheduled-by-a"]);

        assert_eq!(queue.pop_live(|_| true), Some("b"));
        assert_eq!(queue.pop_live(|_| true), Some("scheduled-by-a"));
        assert_eq!(queue.pop_live(|_| true), None);
    }

    #[test]
    fn pop_live_drops_dead_tickets() {
        let mut queue = RedeemQueue::new();
        queue.schedule(["live-1", "dead-1", "dead-2", "live-2"]);

        let is_live = |tx: &&str| tx.starts_with("live");
        assert_eq!(queue.pop_live(is_live), Some("live-1"));
        assert_eq!(queue.pop_live(is_live), Some("live-2"));
        assert!(queue.is_empty());
    }

    #[test]
    fn iter_walks_fifo_without_consuming() {
        let mut queue = RedeemQueue::new();
        queue.schedule(["a", "b"]);

        assert_eq!(queue.iter().collect::<Vec<_>>(), [&"a", &"b"]);
        assert_eq!(queue.len(), 2);
        assert_eq!(queue.pop_live(|_| true), Some("a"));
    }

    #[test]
    fn clear_empties_the_queue() {
        let mut queue = RedeemQueue::new();
        queue.schedule(["a", "b"]);
        queue.clear();
        assert!(queue.is_empty());
        assert_eq!(queue.pop_live(|_| true), None);
    }

    #[test]
    fn pop_live_returns_none_when_all_dead() {
        let mut queue = RedeemQueue::new();
        queue.schedule(["dead-1", "dead-2"]);

        assert_eq!(queue.pop_live(|_| false), None);
        assert!(queue.is_empty());
    }
}
