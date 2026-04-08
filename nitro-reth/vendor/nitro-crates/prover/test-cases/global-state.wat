(import "wrapper" "set_globalstate_u64" (func $set (param i32) (param i64)))
(import "wrapper" "get_globalstate_u64" (func $get (param i32) (result i64)))
(import "wrapper" "halt_and_set_finished" (func $halt))

(func $entry
	(i32.const 0)
	(i64.const 10)
	(call $set)
	(loop
		(i32.const 0)
		(i32.const 0)
		(call $get)
		(i64.sub (i64.const 1))
		(call $set)
		(i32.const 0)
		(call $get)
		(i32.wrap_i64)
		(br_if 0)
	)
	(call $halt)
)

(start $entry)
