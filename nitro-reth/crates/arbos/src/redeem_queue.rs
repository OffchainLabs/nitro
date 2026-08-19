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

    /// Appends a retry scheduled by the tx that just executed.
    pub fn schedule(&mut self, tx: T) {
        self.queue.push_back(tx);
    }

    pub fn clear(&mut self) {
        self.queue.clear();
    }

    /// Iterates the pending retries in FIFO order without consuming them.
    pub fn iter(&self) -> impl Iterator<Item = &T> {
        self.queue.iter()
    }

    /// Pops the next scheduled retry.
    pub fn pop(&mut self) -> Option<T> {
        self.queue.pop_front()
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
        for tx in ["a", "b", "c"] {
            queue.schedule(tx);
        }
        assert_eq!(queue.len(), 3);

        assert_eq!(queue.pop(), Some("a"));
        assert_eq!(queue.pop(), Some("b"));
        assert_eq!(queue.pop(), Some("c"));
        assert_eq!(queue.pop(), None);
        assert!(queue.is_empty());
    }

    #[test]
    fn retries_scheduled_by_a_retry_join_the_back() {
        // Go appends ScheduledTxes after each execution: with [a, b] pending, a retry scheduled
        // while executing "a" runs after "b".
        let mut queue = RedeemQueue::new();
        queue.schedule("a");
        queue.schedule("b");

        assert_eq!(queue.pop(), Some("a"));
        queue.schedule("scheduled-by-a");

        assert_eq!(queue.pop(), Some("b"));
        assert_eq!(queue.pop(), Some("scheduled-by-a"));
        assert_eq!(queue.pop(), None);
    }

    #[test]
    fn iter_walks_fifo_without_consuming() {
        let mut queue = RedeemQueue::new();
        queue.schedule("a");
        queue.schedule("b");

        assert_eq!(queue.iter().collect::<Vec<_>>(), [&"a", &"b"]);
        assert_eq!(queue.len(), 2);
        assert_eq!(queue.pop(), Some("a"));
    }

    #[test]
    fn clear_empties_the_queue() {
        let mut queue = RedeemQueue::new();
        queue.schedule("a");
        queue.schedule("b");
        queue.clear();
        assert!(queue.is_empty());
        assert_eq!(queue.pop(), None);
    }
}
