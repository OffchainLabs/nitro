(func
	(i32.const 1)
	(if (result i32)
		(then
			(i32.const 2)
		)
		(else (unreachable))
	)
	(drop)
	(i32.const 0)
	(if (result i32)
		(then (unreachable))
		(else
			(i32.const 3)
			(br 0)
		)
	)
	(drop)
)

(func (export "user_entrypoint") (param $args_len i32) (result i32)
	(i32.const 0)
)

(start 0)
