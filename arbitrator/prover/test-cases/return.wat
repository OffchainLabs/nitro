(func
	(i32.const 1)
	(call 1)
	(drop)
)

(func (param i32) (result i32)
	(i32.const 5)
	(local.get 0)
	(if (param i32) (result i32)
		(then
			(i32.const 0)
			(call 1)
			(i32.add)
		)
		(else
			(i32.const 10)
			(return)
		)
	)
)

(start 0)

