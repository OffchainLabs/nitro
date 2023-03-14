(module 
  (memory 1)

  (func $main

    i32.const 20 ;; pointer into start of region to fill
    i32.const 55 ;; value to fill
    i32.const 10 ;; length of segment to fill
    memory.fill

    (call $assert_byte_at_address (i32.const 19) (i32.const 0))
    (call $assert_byte_at_address (i32.const 20) (i32.const 55))
    (call $assert_byte_at_address (i32.const 29) (i32.const 55))
    (call $assert_byte_at_address (i32.const 30) (i32.const 0))


    i32.const 300 ;; pointer to destination
    i32.const 20 ;; pointer to source
    i32.const 5 ;; number of bytes to copy
    memory.copy 
    
    (call $assert_byte_at_address (i32.const 300) (i32.const 55))
    (call $assert_byte_at_address (i32.const 304) (i32.const 55))
    (call $assert_byte_at_address (i32.const 305) (i32.const 0))
    

    i32.const 30 ;; pointer into start of region to fill
    i32.const 55 ;; value to fill
    i32.const 0 ;; length of segment to fill
    memory.fill

    (call $assert_byte_at_address (i32.const 30) (i32.const 0))


    i32.const 306 ;; pointer into start of region to fill
    i32.const 71 ;; value to fill
    i32.const 50 ;; length of segment to fill
    memory.fill
    
    (call $assert_byte_at_address (i32.const 305) (i32.const 0))
    (call $assert_byte_at_address (i32.const 306) (i32.const 71))
    (call $assert_byte_at_address (i32.const 320) (i32.const 71))
    (call $assert_byte_at_address (i32.const 351) (i32.const 71))
    (call $assert_byte_at_address (i32.const 355) (i32.const 71))
    (call $assert_byte_at_address (i32.const 356) (i32.const 0))
    ;; memory is currently 55 on [20,29] and [300,304]; 71 on [306, 355]

    ;; goal: do 5 copies, src (> & <) dest with overlapping & not; plus src==dest
    ;; we've already done src < dest nonoverlapping above

    ;; source > dest
    i32.const 50 ;; pointer to destination
    i32.const 304 ;; pointer to source
    i32.const 10 ;; number of bytes to copy
    memory.copy
    
    (call $assert_byte_at_address (i32.const 49) (i32.const 0))
    (call $assert_byte_at_address (i32.const 50) (i32.const 55))
    (call $assert_byte_at_address (i32.const 51) (i32.const 0))
    (call $assert_byte_at_address (i32.const 52) (i32.const 71))
    (call $assert_byte_at_address (i32.const 55) (i32.const 71))
    (call $assert_byte_at_address (i32.const 59) (i32.const 71))
    (call $assert_byte_at_address (i32.const 60) (i32.const 0))
    ;;memory now has [55, 0, 71 * 8] stored at [50,59]

    ;; source < dest, overlapping 
    i32.const 52 ;; pointer to destination
    i32.const 50 ;; pointer to source
    i32.const 9 ;; number of bytes to copy
    memory.copy
    (call $assert_byte_at_address (i32.const 49) (i32.const 0))
    (call $assert_byte_at_address (i32.const 50) (i32.const 55))
    (call $assert_byte_at_address (i32.const 51) (i32.const 0))
    (call $assert_byte_at_address (i32.const 52) (i32.const 55))
    (call $assert_byte_at_address (i32.const 53) (i32.const 0))
    (call $assert_byte_at_address (i32.const 54) (i32.const 71))
    (call $assert_byte_at_address (i32.const 55) (i32.const 71))
    (call $assert_byte_at_address (i32.const 59) (i32.const 71))
    (call $assert_byte_at_address (i32.const 60) (i32.const 71))
    (call $assert_byte_at_address (i32.const 61) (i32.const 0))
    ;; memory now has [55, 0, 55, 0, 71 * 7] stored at [50,60]

    ;; source == dest
    i32.const 50 ;; pointer to destination
    i32.const 50 ;; pointer to source
    i32.const 25 ;; number of bytes to copy
    memory.copy
    (call $assert_byte_at_address (i32.const 49) (i32.const 0))
    (call $assert_byte_at_address (i32.const 50) (i32.const 55))
    (call $assert_byte_at_address (i32.const 51) (i32.const 0))
    (call $assert_byte_at_address (i32.const 52) (i32.const 55))
    (call $assert_byte_at_address (i32.const 53) (i32.const 0))
    (call $assert_byte_at_address (i32.const 55) (i32.const 71))
    (call $assert_byte_at_address (i32.const 59) (i32.const 71))
    (call $assert_byte_at_address (i32.const 60) (i32.const 71))
    (call $assert_byte_at_address (i32.const 61) (i32.const 0))
    (call $assert_byte_at_address (i32.const 74) (i32.const 0))

    ;; source > dest, overlapping
    i32.const 46 ;; pointer to destination
    i32.const 55 ;; pointer to source
    i32.const 7 ;; number of bytes to copy
    memory.copy
    (call $assert_byte_at_address (i32.const 45) (i32.const 0))
    (call $assert_byte_at_address (i32.const 46) (i32.const 71))
    (call $assert_byte_at_address (i32.const 49) (i32.const 71))
    (call $assert_byte_at_address (i32.const 50) (i32.const 71))
    (call $assert_byte_at_address (i32.const 51) (i32.const 71))
    (call $assert_byte_at_address (i32.const 52) (i32.const 0))
    (call $assert_byte_at_address (i32.const 53) (i32.const 0))
    (call $assert_byte_at_address (i32.const 55) (i32.const 71))
    (call $assert_byte_at_address (i32.const 59) (i32.const 71))
    (call $assert_byte_at_address (i32.const 60) (i32.const 71))
    (call $assert_byte_at_address (i32.const 61) (i32.const 0))
    ;; memory now has [71 * 6, 0, 0, 71 * 7] stored at [46, 60]

    ;; length 0
    i32.const 50 ;; pointer to destination
    i32.const 52 ;; pointer to source
    i32.const 0 ;; number of bytes to copy
    memory.copy
    (call $assert_byte_at_address (i32.const 50) (i32.const 71))
    (call $assert_byte_at_address (i32.const 52) (i32.const 0))
  )

  (func $assert_byte_at_address (param i32) (param i32)
    (local.get 0) 
    i32.load8_u ;; load8u to load a byte as unsigned, which is also how the memory ops work
    (local.get 1) 
    i32.eq
    (call $assert_true)
  )

  (func $assert_true (param i32)
    (local.get 0)
    i32.eqz
    (if
      (then
        (local.get 0)
        unreachable
      )
    )
  )

  (start $main)
)