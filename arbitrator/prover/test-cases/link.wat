;; Copyright 2023, Offchain Labs, Inc.
;; For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

(module
    (import "hostio" "wavm_link_module" (func $link (param i32) (result i32)))
    (import "hostio" "wavm_unlink_module" (func $unlink (param) (result)))
    (data (i32.const 0x000)
        "\b8\ef\6c\9d\e3\a2\aa\9a\94\56\09\7b\13\ef\19\ab\82\ab\c6\d2\eb\76\ca\72\9a\dd\b8\20\84\7b\7a\f7") ;; block
    (data (i32.const 0x020)
        "\e0\7a\1c\2e\5f\26\ab\ff\5b\ef\d4\ae\b4\c1\7a\18\52\06\cd\be\9c\ea\fb\c3\b6\4a\03\4b\c0\c4\51\14") ;; call
    (data (i32.const 0x040)
        "\4c\85\59\6e\e9\89\38\42\7e\52\63\f5\98\c9\fe\23\4a\17\11\d8\37\e3\ed\d4\2a\93\e7\32\15\c2\d7\1c") ;; indirect
    (data (i32.const 0x060)
        "\f9\0f\a0\15\f7\e0\52\c2\c8\ec\ac\1c\b7\a3\58\8c\00\52\52\95\a1\ec\a0\55\90\74\aa\11\3f\4e\75\55") ;; const
    (data (i32.const 0x080)
        "\2d\e2\b0\dd\15\63\31\27\e5\31\2d\2d\23\e4\13\67\a0\d5\4d\0e\66\d9\2d\79\37\5d\44\7c\d0\80\f8\b0") ;; div
    (data (i32.const 0x0a0)
        "\91\7a\9d\9d\03\77\07\f0\f7\84\b4\f1\01\e5\50\ff\5f\40\13\34\c1\af\c1\f2\f6\8f\f7\03\a3\ba\32\47") ;; globals
    (data (i32.const 0x0c0)
        "\31\81\c9\76\80\55\57\40\6d\93\0d\46\3b\60\31\de\4b\0f\93\14\8e\78\58\63\8c\66\88\55\c3\d3\47\b2") ;; if-else
    (data (i32.const 0x0e0)
        "\8d\e8\a2\29\d8\22\25\97\3e\57\7f\17\2f\b0\24\94\3f\fe\73\6b\ef\34\18\10\4b\18\73\87\4c\3d\97\88") ;; locals
    (data (i32.const 0x100)
        "\6a\0b\f4\9a\9f\c4\68\18\a5\23\79\11\bf\3a\8d\02\72\ea\e1\21\db\b8\f3\a5\72\d3\66\9a\27\f1\01\74") ;; loop
    (data (i32.const 0x120)
        "\35\fb\41\8d\94\da\56\dc\3b\2e\8f\6c\7f\43\bf\dd\9a\30\7c\27\b9\b2\e0\dd\7d\15\29\36\53\ca\b7\77") ;; math
    (data (i32.const 0x140)
        "\26\d0\56\ce\4a\fa\f1\ef\cd\d2\7a\4d\64\48\a3\86\5a\00\5f\6f\89\7e\a4\95\5c\b9\2d\b8\f1\04\eb\3e") ;; iops
    (data (i32.const 0x160)
        "\dd\13\30\ba\64\c6\e1\e9\e9\c3\b0\f9\63\c8\d9\69\c9\f0\13\70\42\17\f3\9e\60\c3\0d\13\5b\48\a5\88") ;; user
    (data (i32.const 0x180)
        "\07\33\43\e0\1d\b9\16\3e\99\1a\99\bd\cc\93\bf\26\15\f4\4c\11\c3\32\c0\2c\65\ba\76\11\76\eb\c1\7b") ;; return
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
             br_if $top))
    (memory 1)
    (start $start))
