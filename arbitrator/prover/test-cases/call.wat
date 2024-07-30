(func
	(i32.const 1)
	(call 1)
)

(func (param i32)
	(local.get 0)
	(if
		(then (call 2))
		(else (unreachable))
	)
)

(func
	(i32.const 0)
	(call 1)
)

(func (export "user_entrypoint") (param $args_len i32) (result i32)
	(i32.const 0)
)

(start 0)
(memory (export "memory") 0 0)
