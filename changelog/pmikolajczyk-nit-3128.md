### Removed
- Remove `MarkValid` method from the `ExecutionRecorder` interface. We run the old `BlockRecorder.MarkValid` logic from `ExecutionNode.SetFinalityData`.