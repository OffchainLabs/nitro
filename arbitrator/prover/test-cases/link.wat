;; Copyright 2023, Offchain Labs, Inc.
;; For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

(module
    (import "hostio" "wavm_link_module"        (func $link (param i32) (result i32)))
    (import "hostio" "wavm_unlink_module"      (func $unlink (param) (result)))
    (import "env" "wavm_halt_and_set_finished" (func $halt                         ))

    ;; WAVM module hashes
    (data (i32.const 0x000)
        "\8a\e6\f4\76\01\53\24\95\60\68\9c\8c\f4\31\02\bb\3a\c7\9c\19\06\2b\b6\a3\aa\f6\16\71\3c\d6\21\ca") ;; block
    (data (i32.const 0x020)
        "\6a\66\98\e6\88\ec\a0\d0\dd\36\96\27\00\0a\0f\d9\c0\0c\90\26\c2\d4\c1\7d\c5\d5\c5\ff\06\51\7d\c5") ;; call
    (data (i32.const 0x040)
        "\be\0d\fd\02\ad\b9\a0\c1\d9\bc\1e\35\c4\d5\0f\bf\bb\84\30\ff\9a\66\e1\1e\1f\d4\7c\4a\2f\43\61\72") ;; call-indirect
    (data (i32.const 0x060)
        "\93\34\7f\c7\0e\62\c1\96\c0\15\2c\da\30\32\06\47\e4\d3\b5\73\8f\e4\b5\29\02\dc\87\f0\0e\a3\c9\0f") ;; const
    (data (i32.const 0x080)
        "\43\2c\ee\07\43\5b\66\e9\31\81\05\cf\ce\99\95\c2\62\00\96\92\79\9e\d1\5e\22\da\7b\3c\28\f5\f6\20") ;; div-overflow
    (data (i32.const 0x0a0)
        "\fb\58\be\58\45\59\b4\3c\3e\68\d8\fb\09\90\db\ab\f9\a4\c9\e2\e0\4a\bb\ef\97\c4\8a\6c\63\66\98\10") ;; globals
    (data (i32.const 0x0c0)
        "\ba\6f\20\22\c0\90\b8\9f\10\14\bd\24\73\15\b3\85\b7\67\83\75\db\24\9c\aa\b2\d7\0d\20\39\de\cf\1d") ;; if-else
    (data (i32.const 0x0e0)
        "\f3\0a\be\d6\b9\c7\fe\81\c3\0e\95\f3\d8\d2\5f\67\b0\a2\11\89\b4\ea\77\c8\f6\c0\f8\6f\0e\04\0b\8d") ;; locals
    (data (i32.const 0x100)
        "\82\e6\f6\50\86\e2\cb\d7\3c\18\cb\f8\34\89\1c\16\b7\fe\ea\26\5d\55\9c\d0\c7\8b\1e\1f\d5\6a\6f\14") ;; loop
    (data (i32.const 0x120)
        "\10\7f\1c\0d\eb\d2\8a\4f\24\f2\f4\55\b0\f2\73\25\b7\db\70\5d\71\6b\40\70\e8\00\94\ac\29\e0\b2\09") ;; math
    (data (i32.const 0x140)
        "\4b\b1\93\f1\b8\4e\f1\72\7b\80\63\b3\28\6c\45\52\3f\06\d2\15\f4\6d\a1\ca\32\a6\2e\3e\5c\4b\92\ff") ;; iops
    (data (i32.const 0x160)
        "\a4\73\76\c8\ea\84\f2\58\06\c6\17\83\a4\c1\a0\18\ab\72\5c\8c\03\53\95\db\91\6b\29\ec\3a\b9\43\14") ;; user
    (data (i32.const 0x180)
        "\53\f0\be\e8\1d\fb\ba\b6\29\54\fa\73\45\08\3c\cd\a0\5f\c9\39\90\fd\ba\da\bc\3a\54\9e\56\37\73\58") ;; return

    (func $start (local $counter i32)

         ;; add modules
         (loop $top
             ;; increment counter
             local.get $counter
             local.get $counter
             i32.const 1
             i32.add
             local.set $counter

             ;; link module with unique hash
             i32.const 32
             i32.mul
             call $link

             ;; loop until 12 modules
             i32.const 12
             i32.le_s
             br_if $top
         )

         ;; reset counter
         i32.const 0
         local.set $counter

         ;; link and unlink modules
         (loop $top
             ;; increment counter
             local.get $counter
             local.get $counter
             i32.const 1
             i32.add
             local.set $counter

             ;; unlink 2 modules
             call $unlink
             call $unlink

             ;; link module with unique hash
             i32.const 32
             i32.mul
             call $link

             ;; loop until most are gone
             i32.const 3
             i32.ge_s
             br_if $top)

        call $halt
    )
    (memory 1)
    (start $start))
