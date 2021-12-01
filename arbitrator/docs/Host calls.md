# Host calls

Describes the implementation of various functions WASM programs can import which are implemented by the host environment.

## Overview

| Module                 | Name                | Function signature |
|------------------------|---------------------|--------------------|
| wasi_snapshot_preview1 | environ_sizes_get   | (i32, i32) -> i32
| wasi_snapshot_preview1 | environ_get         | (i32, i32) -> i32
| wasi_snapshot_preview1 | proc_exit           | (i32)
| env                    | exit                | (i32)
| wasi_snapshot_preview1 | fd_write            | (i32, i32, i32, i32) -> i32
| wasi_snapshot_preview1 | fd_close            | (i32) -> i32
| wasi_snapshot_preview1 | fd_read             | (i32) -> i32
| wasi_snapshot_preview1 | path_open           | (i32, i32, i32, i32, i32, i64, i64, i32, i32) -> i32
| wasi_snapshot_preview1 | random_get          | (i32, i32) -> i32
| wasi_snapshot_preview1 | fd_prestat_get      | (i32, i32) -> i32
| wasi_snapshot_preview1 | fd_prestat_dir_name | (i32, i32, i32) -> i32

## Implementations

### environ_sizes_get
Sets the values of both argument pointers to 0, indicating no environment variables are present, then returns 0, indicating no error.

### environ_get
Returns errno 28 "Invalid argument", as no environment variables are present.

### proc_exit
Halts the machine.

### fd_write
As per the WASI spec, totals the size of the input "ciovec"s and writes it to the last pointer, then returns 0.

### fd_close
Returns errno 8: BADFD

### fd_read
Returns errno 8: BADFD

### path_open
Halts the machine

### random_get
Returns 0, indicating success, but fills in no data (TODO: should probably still fill in something)

### fd_prestat_get
Returns errno 8: BADFD

### fd_prestat_dir_name
Returns errno 8: BADFD
