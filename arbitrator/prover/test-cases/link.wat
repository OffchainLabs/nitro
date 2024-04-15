;; Copyright 2023, Offchain Labs, Inc.
;; For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

(module
    (import "hostio" "wavm_link_module"        (func $link (param i32) (result i32)))
    (import "hostio" "wavm_unlink_module"      (func $unlink (param) (result)))
    (import "env" "wavm_halt_and_set_finished" (func $halt                         ))

    ;; WAVM module hashes
        (data (i32.const 0x000)
        "\56\19\01\5f\5d\d4\1f\5a\f8\39\eb\a7\71\a5\8e\e8\a4\d1\3a\dd\ee\2e\75\29\9a\19\cc\89\a5\ab\d3\73") ;; block
    (data (i32.const 0x020)
        "\7a\02\20\3c\1a\93\f8\0a\7c\1c\43\b3\95\79\c5\9d\f7\c3\84\5d\be\2e\1a\9d\6f\58\88\87\c0\a2\fe\13") ;; call
    (data (i32.const 0x040)
        "\76\aa\58\26\ed\70\37\00\01\c1\f0\62\4c\cb\23\77\1e\03\a0\e7\34\a8\45\11\c3\bd\de\4e\03\40\4a\5c") ;; indirect
    (data (i32.const 0x060)
        "\79\54\72\df\45\56\6f\2f\5f\85\06\60\ec\3b\0a\43\ce\f0\3b\90\75\7d\86\82\d1\8d\c1\fe\da\31\40\bb") ;; const
    (data (i32.const 0x080)
        "\9e\48\3c\16\fb\ec\9b\90\de\34\8f\38\26\a7\41\44\0a\fb\1c\21\f4\e3\76\be\a2\f3\d7\03\4a\1d\9c\a2") ;; div
    (data (i32.const 0x0a0)
        "\38\cb\94\a1\4d\d1\ab\9a\29\b0\f7\5e\c7\f0\cb\db\1d\f5\fe\34\52\8e\26\7a\25\c8\a8\8e\d4\a4\16\f9") ;; globals
    (data (i32.const 0x0c0)
        "\36\62\29\c5\f3\d2\3e\8e\21\02\8d\ef\95\04\2d\d8\a5\1b\08\2d\30\d7\6b\6c\85\83\4b\19\be\8e\dd\ba") ;; if-else
    (data (i32.const 0x0e0)
        "\98\5d\8a\d6\ac\09\6b\bd\cc\ca\7c\87\a9\20\db\11\5f\b1\28\e1\a1\51\70\8a\9f\46\bf\f0\f8\c8\d0\e2") ;; locals
    (data (i32.const 0x100)
        "\9a\cc\60\ec\96\44\53\09\1c\0c\2e\19\42\f2\b4\db\56\a7\d4\40\2e\36\f3\03\33\43\05\de\ea\c5\6b\47") ;; loop
    (data (i32.const 0x120)
        "\2b\d8\a0\ed\09\1c\47\03\b1\55\d7\a6\b0\bd\24\68\e0\0b\92\a6\b8\fe\2c\71\b4\c7\bf\40\05\6d\f4\2d") ;; math
    (data (i32.const 0x140)
        "\7e\01\98\c8\a1\f4\74\be\92\8c\2c\ec\5d\5f\be\04\65\b1\c0\74\43\71\c3\63\00\db\20\b3\a9\17\9b\ac") ;; iops
    (data (i32.const 0x160)
        "\3a\eb\a0\67\68\ef\f5\f9\4e\ec\84\88\ac\54\b7\b7\07\a5\12\9c\fb\73\50\37\33\d9\9e\90\ea\72\97\8c") ;; user
    (data (i32.const 0x180)
        "\fa\91\57\09\98\8a\54\d2\d5\96\71\13\da\71\ae\80\eb\b1\b3\68\5e\90\d7\8e\0e\7d\a2\c4\d8\d9\72\cf") ;; return

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
