(global (mut i32) (i32.const 1))
(global (mut i32) (i32.const 2))

(func
	(global.get 0)
	(i32.const 1)
	(i32.add)
	(global.set 0)
	(global.get 0)
	(global.set 1)
	(global.get 0)
	(i32.const 1)
	(i32.add)
	(global.set 1)
	(global.get 0)
	(global.get 1)
	(i32.add)
	(drop)
)

(start 0)



