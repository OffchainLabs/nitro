# WASM to WAVM

Not all WASM instructions are 1:1 with WAVM opcodes.
This document lists those which are not, and explains how they're expressed in WAVM.
Many of the WAVM representations use opcodes not in WASM,
which are documented in `WAVMCustomOpcodes.md`.

## `block`

In WASM, a block contains instructions.
There's no such concept of containing instructions in WAVM, which is linear.
A WASM block is expressed in WAVM as a block instruction with argument data pointing to after the end of the block,
then the instructions inside the block, and then an EndBlock instruction.

## `loop`

This is the same as `block`, except the generated `block` instruction's argument data points to itself.

## `if` and `else`

These are translated to a block with an `ArbitraryJumpIf` as follows:

```
begin block with endpoint end
  conditional jump to else
  [instructions inside if statement]
  branch
  else: [instructions inside else statement]
end
```

## `br`

`br x` is translated to `x` WAVM `EndBlock` instructions and then a `Branch`.

## `br_if`

`br_if x` is translated to `x` WAVM `EndBlockIf` instructions and then a `BranchIf`.

## `br_table`

`br_table` is translated to a check for each possible branch in the table,
and then if none of the checks hit, a branch of the default level.

Each of the non-default branches has a conditional jump to a branch that far,
which is put after the default branch in code.

## `local.tee`

`local.tee` is translated to a WAVM `Dup` and then a `LocalSet`.

## `return`

To translate a return, the number of return values must be known from the function signature.
A WAVM `MoveFromStackToInternal` is added for each return value.
Then, a loop checks `IsStackBoundary` (which implicitly pops a value) until it's true and the stack boundary has been popped.
If the return is nested inside blocks, an `EndBlock` is generated for each one.
Next, a `MoveFromInternalToStack` is added for each return value to put the return values back on the stack.
Finally, a WAVM `Return` is added, returning control flow to the caller.

## Floating point instructions

A floating point library module must be present to translate floating point instructions.
They are translated by bitcasting `f32` and `f64` arguments to `i32`s and `f64`s,
then a cross module call to the floating point library,
and finally bitcasts of any return values from `i32`s and `i64`s to `f32`s and `f64`s.
