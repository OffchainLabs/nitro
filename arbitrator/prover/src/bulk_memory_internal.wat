(memory 1)

;;            length     value to set pointer to start of region
(func $memset (param i32) (param i32) (param i32)
  (local $offset i32)
  (i32.const 0)
  (local.set $offset) 

  (loop $inner
    ;;calculate current index into region to be set
    (local.get $offset)
    (local.get 2) 
    i32.add 
    (local.get 1) 
    (i32.store8)
    ;;increment offset
    (i32.const 1)
    (local.get $offset)
    i32.add 
    (local.tee $offset)
    ;;check to terminate loop 
    (local.get 0) 
    i32.ne
    (br_if $inner)
  )
)

;;            length l    source s    dest d
(func $memcpy (param i32) (param i32) (param i32)
  (local $offset i32) ;; o 

  (local.get 1) 
  (local.get 2) 
  (i32.gt_s)
  (if ;;copy forward when source >= dest
    (then 
      ;;offset starts at 0
      (i32.const 0)
      (local.set $offset) 
      (loop $forward
        ;;put d + o on stack
        (local.get $offset) 
        (local.get 2) 
        i32.add 
        ;;load from s + o
        (local.get $offset)
        (local.get 1) 
        i32.add
        i32.load8_u
        ;;store to d + o 
        i32.store8
        ;; increment offset
        (local.get $offset)
        (i32.const 1) 
        i32.add 
        (local.tee $offset) 
        ;;check to terminate loop 
        (local.get 0) 
        i32.ne
        (br_if $forward)
      ) 
    )
    (else
      ;;offset starts at (l-1)
      (local.get 0) 
      (i32.const 1)
      i32.sub 
      (local.set $offset)
      (loop $backward
        ;;put d + o on stack
        (local.get $offset) 
        (local.get 2) 
        i32.add 
        ;;load from s + o
        (local.get $offset)
        (local.get 1) 
        i32.add
        i32.load8_u
        ;;store to d + o 
        i32.store8
        ;; decrement offset
        (local.get $offset)
        (i32.const 1) 
        i32.sub 
        (local.tee $offset) 
        ;;check to terminate loop 
        (i32.const -1)
        i32.ne
        (br_if $backward)
      ) 
    )
  )
)

(func $main 
  (i32.const 100)
  (i32.const 3)
  (i32.const 2)
  (call $memset)
)

(start $main)