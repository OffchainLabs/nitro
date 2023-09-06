;; Copyright 2023, Offchain Labs, Inc.
;; For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

(module
    (import "hostio" "wavm_link_module" (func $link (param i32) (result i32)))
    (import "hostio" "wavm_unlink_module" (func $unlink (param) (result)))
    (data (i32.const 0x000)
        "\a6\44\c7\fc\d3\a2\7b\00\60\f2\7c\32\47\2c\3b\0f\c0\88\94\8c\5b\9f\b1\9c\17\11\9d\70\04\6e\9e\25") ;; block
    (data (i32.const 0x020)
        "\4f\0f\fa\e9\f1\a2\5b\72\85\9d\c8\23\aa\ed\42\18\54\ed\b1\14\9f\08\97\26\fc\e2\ff\ad\ca\2b\96\bc") ;; call
    (data (i32.const 0x040)
        "\71\4b\0c\ab\49\45\e7\e1\e5\34\83\c7\33\0f\36\6a\29\42\45\a5\91\e0\91\7a\f7\0a\ae\f2\fe\2a\72\b4") ;; indirect
    (data (i32.const 0x060)
        "\fc\ef\2f\e4\98\5c\63\b5\4d\f2\39\86\98\91\c6\70\93\18\d6\22\45\7a\f4\be\fb\ac\34\19\8f\9a\69\3b") ;; const
    (data (i32.const 0x080)
        "\ce\85\04\55\06\33\44\e6\30\3b\14\33\b3\8e\c5\41\ac\bf\96\60\cb\45\47\97\8c\b6\99\6e\ef\76\d1\36") ;; div
    (data (i32.const 0x0a0)
        "\01\05\9b\42\54\f2\80\00\0e\2c\41\ed\79\e3\f5\69\d1\28\e6\d3\4e\f5\20\b9\4d\ee\31\5e\78\a4\6b\3e") ;; globals
    (data (i32.const 0x0c0)
        "\e7\ac\97\8c\df\27\ca\1d\50\30\4d\b4\0c\1f\23\1a\76\bb\eb\5e\2a\2e\5b\e5\4d\24\a4\cc\9d\91\eb\93") ;; if-else
    (data (i32.const 0x0e0)
        "\f3\3e\62\9a\ee\08\b3\4e\cd\15\a0\38\dc\cc\80\71\b0\31\35\16\fb\4e\77\34\c6\4d\77\54\85\38\7f\35") ;; locals
    (data (i32.const 0x100)
        "\1d\c4\11\d8\36\83\4a\04\c0\7b\e0\46\a7\8d\4e\91\0b\13\f2\d5\1a\9e\fe\ed\9d\e6\2f\ee\54\6f\94\95") ;; loop
    (data (i32.const 0x120)
        "\8a\f6\10\f0\c6\a1\91\55\0a\72\1e\4d\36\91\88\6b\18\f5\42\73\9d\c5\9a\ea\1d\4d\b5\fb\bf\cf\06\f0") ;; math
    (data (i32.const 0x140)
        "\fc\27\e9\2e\12\23\f2\d6\ef\2a\83\3b\c8\1a\22\99\77\76\23\d8\f5\cf\51\f8\28\ba\a4\27\98\af\aa\24") ;; iops
    (data (i32.const 0x160)
        "\10\a4\b0\c7\91\26\6b\fb\f7\92\f5\e5\67\e0\03\d7\ee\7f\cf\7e\0a\52\6e\b3\92\46\c3\94\6f\21\b8\f8") ;; user
    (data (i32.const 0x180)
        "\f6\ad\69\79\fc\db\8a\af\27\48\ac\7c\54\5f\b2\a8\f2\80\f8\69\a6\75\59\a7\80\58\ba\26\39\5e\aa\c9") ;; return
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
