# WAVM Custom opcodes not in WASM

## Codegen internal

These are generated when breaking down a WASM instruction that does many things into many WAVM instructions which each do one thing.
For instance, `local.tee` is implemented with `dup` and then `local.set`, the former of which doesn't exist in WASM.

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

## Linking

This is only generated to link modules together.
Each import is replaced with a local function consisting primarily of this opcode,
which handles the actual work needed to change modules.

| Opcode | Name            | Description |
|--------|-----------------|-------------|
| 0x8009 | CrossModuleCall |

## Host calls

These are only used in the implementation of "host calls".
Each of these has an equivalent host call method, which can be invoked from libraries.
The exception is `CallerModuleInternalCall`,
which is used for the implementation of all of the `wavm_caller_*` host calls.

| Opcode | Name                     | Description |
|--------|--------------------------|-------------|
| 0x800A | CallerModuleInternalCall |
| 0x8010 | GetGlobalStateBytes32    |
| 0x8011 | SetGlobalStateBytes32    |
| 0x8012 | GetGlobalStateU64        |
| 0x8013 | SetGlobalStateU64        |
| 0x8020 | ReadPreImage             |
| 0x8021 | ReadInboxMessage         |
| 0x8022 | HaltAndSetFinished       |
