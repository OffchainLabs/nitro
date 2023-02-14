(memory 1)

(data (i32.const 1) "\01\02")
(table $firsttable 2 funcref)
(elem (table $firsttable) (i32.const 0) $f1 $f2)
(table $secondtable 2 funcref)
(elem (table $secondtable) (i32.const 0) $f3 $f4)

  (func $f1 (result i32)
    i32.const 42)
  (func $f2 (result i32)
    i32.const 13)
  (func $f3 (result i32)
    i32.const 43)
  (func $f4 (result i32)
    i32.const 14)

(type $return_i32 (func (result i32)))

(func $main

  ;; (i32.const 20) ;; offset
  ;; (i32.const 5) ;; value to be stored
  ;; (i32.store offset=30)
  ;; (i32.const 21) ;; offset
	;; (i32.load offset=29)
  ;; (i32.eq (i32.const 5))
  ;; (call $assert_true)

  (i32.const 200) ;; pointer into start of region to fill
  (i32.const 7) ;; value to fill
  (i32.const 10) ;; length of segment to fill
  (memory.fill)

  (call $assert_byte_at_address (i32.const 209) (i32.const 7))
  (call $assert_byte_at_address (i32.const 300) (i32.const 0))

  (i32.const 300) ;; pointer to destination
  (i32.const 200) ;; pointer to source
  (i32.const 5) ;; number of bytes to copy
  (memory.copy) 
  
  (call $assert_byte_at_address (i32.const 300) (i32.const 7))
  (call $assert_byte_at_address (i32.const 304) (i32.const 7))
  (call $assert_byte_at_address (i32.const 305) (i32.const 0))
  
  
  ;; (call $f2)
  
  
  (call_indirect $firsttable (type $return_i32) (i32.const 0))
  (i32.eq (i32.const 42))
  (call $assert_true)

  (call_indirect $firsttable (type $return_i32) (i32.const 1))
  (i32.eq (i32.const 13))
  (call $assert_true)

  (call_indirect $secondtable (type $return_i32) (i32.const 0))
  (i32.eq (i32.const 43))
  (call $assert_true)

  (i32.const 0) ;; destination offset
  (i32.const 1) ;; source offset 
  (i32.const 1) ;; length to copy
  (table.copy $secondtable $secondtable)

  (call_indirect $secondtable (type $return_i32) (i32.const 0))
  (i32.eq (i32.const 14))
  (call $assert_true)

  (i32.const 0) ;; destination offset
  (i32.const 1) ;; source offset 
  (i32.const 1) ;; length to copy
  (table.copy $firsttable $secondtable)

  (call_indirect $secondtable (type $return_i32) (i32.const 0))
  (i32.eq (i32.const 14))
  (call $assert_true)


  (call_indirect $firsttable (type $return_i32) (i32.const 0))
  (i32.eq (i32.const 14))
  (call $assert_true)

  ;; (call_indirect $firsttable (type $return_i32) (i32.const 0))
  ;; (i32.eq (i32.const 13))
  ;; (call $assert_true)

  ;; (call_indirect $firsttable (type $return_i32) (i32.const 1))
  ;; (i32.eq (i32.const 13))
  ;; (call $assert_true)

)

(func $assert_byte_at_address (param i32) (param i32)
  (local.get 0) 
  (i32.load8_u) ;;load8u to load a byte as unsigned, which is also how the memory ops work
  (local.get 1) 
  (i32.eq) 
  (call $assert_true)
)

(func $assert_true (param i32)
	(local.get 0)
	(i32.eqz)
	(if
		(then
			(local.get 0)
			unreachable
		)
	)
)

(func $assert_false (param i32)
	(local.get 0)
	(if
		(then
			(local.get 0)
			unreachable
		)
	)
)

(start $main)