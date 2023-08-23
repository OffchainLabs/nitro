(func (local i32)
	(i32.const 0)
	(loop
		(i64.const 123)
		(local.get 0)
		(i32.add (i32.const 1))
		(local.tee 0)
		(i32.ne (i32.const 10))
		(br_if 0)
		(drop)
	)
	(br_if 0)

	(block
		(i32.const 0)
		(loop (param i32)
			(br_if 1)
			(i64.const 123)
			(i32.const 1)
			(br 0)
		)
	)

	(i32.const 1)
	(loop
		(br 0)
		(unreachable)
	)
	(unreachable)
)

(func (export "user_entrypoint") (param $args_len i32) (result i32)
	(i32.const 0)
)

(start 0)


