# Custom opcodes not in WASM

| Opcode | Name                    | Description |
|--------|-------------------------|-------------|
| 0x8000 | EndBlock                | Pops an item from the block stack.
| 0x8001 | EndBlockIf              | Peeks the top value on the stack, assumed an i32. If non-zero, pops an item from the block stack.
| 0x8002 | InitFrame               | Pops a return InternalRef from the stack, and creates a stack frame with the locals merkle root in proving argument data.
| 0x8003 | ArbitraryJumpIf         | Pops an i32 from the stack. If non-zero, jumps to the program counter in the argument data.
| 0x8004 | PushStackBoundary       | Pushes a stack boundary to the stack.
| 0x8005 | MoveFromStackToInternal | Pops an item from the stack and pushes it to the internal stack.
| 0x8006 | MoveFromInternalToStack | Pops an item from the internal stack and pushes it to the stack.
| 0x8007 | IsStackBoundary         | Pops an item from the stack. If a stack boundary, pushes an i32 with value 1. Otherwise, pushes an i32 with value 0.
| 0x8008 | Dup                     | Peeks an item from the stack and pushes another copy to the stack.
| 0x8009 | HostCallHook            | A no-op, which is used by the emulator to introspect host calls
