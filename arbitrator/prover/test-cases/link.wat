;; Copyright 2023, Offchain Labs, Inc.
;; For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

(module
    (import "hostio" "wavm_link_module"        (func $link (param i32) (result i32)))
    (import "hostio" "wavm_unlink_module"      (func $unlink (param) (result)))
    (import "env" "wavm_halt_and_set_finished" (func $halt                         ))

    ;; WAVM module hashes
    (data (i32.const 0x000)
        "\48\39\04\38\2e\ed\a7\eb\5d\d8\9c\a6\36\32\be\84\4b\49\04\d2\6c\09\4b\01\ad\b7\86\47\bb\38\fd\9f") ;; block
    (data (i32.const 0x020)
        "\b2\c0\6d\5c\0b\64\ea\72\14\8e\6b\db\ca\09\0c\62\c5\69\a1\bb\ac\d6\2f\be\92\a3\7e\bd\33\fb\12\90") ;; call
    (data (i32.const 0x040)
        "\d2\e0\77\3e\d1\d4\bc\49\30\0f\fc\06\0a\f0\e4\a4\4f\5f\36\78\56\f4\a9\d7\b5\87\6e\08\be\96\55\4a") ;; indirect
    (data (i32.const 0x060)
        "\3b\e7\f0\69\8e\fc\d0\08\02\13\94\e0\04\d4\60\79\3c\f1\50\ca\84\cb\d8\7a\fe\fc\f1\67\c7\eb\86\79") ;; const
    (data (i32.const 0x080)
        "\89\ed\ef\41\c4\18\2c\7c\3d\56\7c\c9\a2\63\e0\75\20\d1\23\98\87\41\ca\35\75\4f\2c\94\43\b7\11\c2") ;; div
    (data (i32.const 0x0a0)
        "\eb\3f\0e\4d\89\5f\de\9f\02\f0\cd\10\0f\56\9f\d0\7d\71\71\f7\ad\87\95\94\51\7d\47\2b\ea\07\dc\01") ;; globals
    (data (i32.const 0x0c0)
        "\11\07\9f\04\30\99\a1\38\9c\d0\22\b3\00\34\69\b1\3e\ba\46\a8\ff\fd\7e\a6\11\6e\4c\be\aa\f2\1e\36") ;; if-else
    (data (i32.const 0x0e0)
        "\3c\8e\e1\09\a8\94\98\8d\80\43\0c\13\44\0d\0c\d0\45\87\58\8b\8d\ee\2c\11\34\38\fc\c6\e9\39\00\97") ;; locals
    (data (i32.const 0x100)
        "\a4\ae\57\33\6d\a3\1b\50\fe\ca\51\4e\95\d9\65\7a\09\dc\3a\c2\80\24\fd\e3\40\56\fb\94\3a\a4\fe\43") ;; loop
    (data (i32.const 0x120)
        "\e4\12\a2\a6\7e\f8\00\ba\02\4a\38\5f\8e\54\4d\6a\cb\71\61\6d\5d\3a\fe\2f\f8\5c\36\ca\1c\b1\46\cc") ;; math
    (data (i32.const 0x140)
        "\9a\3e\7f\b6\6b\f5\37\32\cd\35\c9\49\6b\cf\42\e1\82\ed\50\4f\bb\20\27\b1\19\2b\01\be\82\76\b4\03") ;; iops
    (data (i32.const 0x160)
        "\54\ee\14\37\13\ef\86\55\20\e8\39\ee\b7\ab\fa\75\3d\32\78\83\6c\b3\32\2c\36\37\57\dd\30\63\c7\24") ;; user
    (data (i32.const 0x180)
        "\05\0a\0c\e4\4d\0f\31\3f\31\d5\df\9c\04\37\82\8c\d2\5b\fe\26\65\ab\e5\55\94\8c\6d\b9\16\4f\64\d4") ;; return

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
