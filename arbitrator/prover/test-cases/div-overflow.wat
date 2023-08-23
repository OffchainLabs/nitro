(func
	(i32.const -0x80000000)
	(i32.const -1)
	(i32.div_s)
	(drop)
	(i64.const -0x8000000000000000)
	(i64.const -1)
	(i64.div_s)
	(drop)
)

(func (export "user_entrypoint") (param $args_len i32) (result i32)
	(i32.const 0)
)

(start 0)
