(func
	(i32.const 1)
	(block
		(block
			(br 1)
			(unreachable)
		)
		(unreachable)
	)
	(block (param i32)
		(br_if 0)
		(unreachable)
	)
	(block
		(block
			(i32.const 2)
			(br_table 0 0 1 0 0)
			(unreachable)
		)
		(unreachable)
	)
	(block
		(block
			(i32.const 8)
			(br_table 0 0 0 0 1)
			(unreachable)
		)
		(unreachable)
	)
)

(start 0)
