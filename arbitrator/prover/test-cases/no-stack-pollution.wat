(import "env" "wavm_halt_and_set_finished" (func $wavm_halt_and_set_finished))

(func
	(i32.const 1)
	(block
		(block
			(i32.const 2)
			(br_table 0 0 1 0 0)
			(unreachable)
		)
		(unreachable)
	)
	(if (i32.eq (i32.const 1))
		(then (call $wavm_halt_and_set_finished))
		(else (unreachable))
	)
)

(start 1)

