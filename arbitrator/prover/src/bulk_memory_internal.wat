(module 
  (memory 1)

  (func $memset (param $pointer i32) (param $value i32) (param $length i32)
    (local $offset i32)
    i32.const 0
    local.set $offset

    (loop $inner
      ;; calculate current index into region to be set
      local.get $offset
      local.get $pointer
      i32.add 
      local.get $value
      i32.store8

      ;; increment offset
      i32.const 1
      local.get $offset
      i32.add 
      local.tee $offset

      ;; check to terminate loop 
      local.get $length
      i32.ne
      br_if $inner
    )
  )

  (func $memcpy (param $destination i32) (param $source i32) (param $length i32)
    (local $offset i32) 

    local.get $source
    local.get $destination
    i32.gt_s
    (if ;; copy forward when source >= dest
      (then 
        ;; offset starts at 0
        i32.const 0
        local.set $offset
        (loop $forward
          ;; put d + o on stack
          local.get $offset
          local.get $destination
          i32.add 
          ;;load from s + o
          local.get $offset
          local.get $source 
          i32.add
          i32.load8_u
          ;;store to d + o 
          i32.store8
          ;; increment offset
          local.get $offset
          i32.const 1
          i32.add 
          local.tee $offset 
          ;;check to terminate loop 
          local.get $length
          i32.ne
          br_if $forward
        ) 
      )
      (else
        ;;offset starts at (l-1)
        local.get $length
        i32.const 1
        i32.sub 
        local.set $offset
        (loop $backward
          ;; put d + o on stack
          local.get $offset
          local.get $destination
          i32.add 

          ;; load from s + o
          local.get $offset
          local.get $source 
          i32.add
          i32.load8_u
          
          ;; store to d + o 
          i32.store8

          ;; decrement offset
          local.get $offset
          i32.const 1
          i32.sub 
          local.tee $offset

          ;; check to terminate loop 
          i32.const -1
          i32.ne
          br_if $backward
        ) 
      )
    )
  )
)
;; (func $main 
;;   (i32.const 100)
;;   (i32.const 3)
;;   (i32.const 2)
;;   (call $memset)
;; )

;; (start $main)