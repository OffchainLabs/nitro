(func
	(nop)
	(i32.const 0)
	(i32.const 1)
	(i32.const 2)
	(drop)
	(drop)
	(drop)
)

(func (export "user_entrypoint") (param $args_len i32) (result i32)
	(i32.const 0)
)

(start 0)
