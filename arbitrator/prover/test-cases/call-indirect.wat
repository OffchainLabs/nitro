(table 4 funcref)
(elem (i32.const 1) 1 2)

(type (func (param i32) (result i32)))
(type (func (param i32)))

(func
	(i32.const 4)
	(i32.const 1)
	(call_indirect (type 0))
	(i32.const 2)
	(call_indirect (type 0))
	(i32.const 1)
	(call_indirect (type 1))
)

(func (param i32) (result i32)
	(local.get 0)
	(i32.const 1)
	(i32.add)
)

(func (param i32) (result i32)
	(local.get 0)
	(i32.const 2)
	(i32.mul)
)

(start 0)
(memory 0 0)
