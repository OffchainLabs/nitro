# Host calls

Describes the implementation of various functions WASM programs can import which are implemented by the host environment.
Each is given a unique ID for the HostCallHook opcode.

## Overview

| ID | Module                 | Name              | Function signature |
|----|------------------------|-------------------|--------------------|
|  0 | wasi_snapshot_preview1 | environ_sizes_get | (i32, i32) -> i32
|  1 | wasi_snapshot_preview1 | environ_get       | (i32, i32) -> i32
|  2 | wasi_snapshot_preview1 | proc_exit         | (i32)
|  2 | env                    | exit              | (i32)
|  3 | wasi_snapshot_preview1 | fd_write          | (i32, i32, i32, i32) -> i32

## Implementations

### environ_sizes_get
Sets the values of both argument pointers to 0, indicating no environment variables are present, then returns 0, indicating no error.

### environ_get
Pops both arguments, then returns errno 28 "Invalid argument", as no environment variables are present.

### proc_exit
Pops its argument, then halts the machine.

### fd_write
Pops 4 arguments. As per the WASI spec, totals the size of the input "ciovec"s and writes it to the last pointer, then returns 0.
