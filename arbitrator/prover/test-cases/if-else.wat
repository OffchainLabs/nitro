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

(start 0)
(memory 0 0)
