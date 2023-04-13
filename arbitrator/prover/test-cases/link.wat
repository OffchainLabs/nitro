;; Copyright 2023, Offchain Labs, Inc.
;; For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

(module
    (import "hostio" "link_module" (func $link (param i32) (result i32)))
    (import "hostio" "unlink_module" (func $unlink (param) (result)))
    (data (i32.const 0x000)
        "\43\d3\30\e2\ab\84\78\27\0d\bc\3c\61\d2\35\2e\c4\86\c8\db\d9\81\e5\8b\8b\ce\19\a7\9d\7b\52\9d\b9") ;; block
    (data (i32.const 0x020)
        "\79\54\4b\1d\46\67\53\83\ab\2f\5b\f4\67\53\27\40\f0\dd\7b\73\11\db\13\4a\01\25\cc\6e\21\12\b5\d5") ;; call
    (data (i32.const 0x040)
        "\e7\5b\ab\eb\38\0f\5e\f6\f9\6c\70\9b\21\c4\ae\c5\e7\a2\32\f7\da\f1\8b\73\bb\af\35\7e\63\24\db\94") ;; indirect
    (data (i32.const 0x060)
        "\5e\2d\9d\d4\bc\e6\15\c7\b5\da\dc\33\a4\c2\6f\b8\52\ca\e4\bd\83\38\89\2e\61\e1\98\81\bc\57\36\dc") ;; const
    (data (i32.const 0x080)
        "\2a\44\34\8e\93\f7\6a\b8\b9\1c\c7\53\e6\1e\1e\10\f1\82\85\ae\7f\e2\0a\0e\bb\e9\8f\ce\c8\7c\ed\37") ;; div
    (data (i32.const 0x0a0)
        "\0f\ae\6d\e4\d4\29\c8\ba\68\6f\b2\36\b3\4f\e7\10\fa\13\64\8e\e3\dc\30\e1\a0\68\60\68\48\93\eb\70") ;; globals
    (data (i32.const 0x0c0)
        "\5a\95\9a\d5\94\8d\03\04\25\a0\6e\5c\71\c3\eb\16\e7\07\50\f8\26\6a\62\6f\ae\ec\33\cd\d2\db\67\4e") ;; if-else
    (data (i32.const 0x0e0)
        "\e4\a9\6f\ca\25\39\c8\83\cc\10\4c\cc\dc\89\9a\3b\2f\20\db\c7\c9\d2\10\d8\3d\97\75\3a\2c\4a\07\db") ;; locals
    (data (i32.const 0x100)
        "\ea\95\a9\54\7b\99\d2\55\6b\a1\2f\6b\39\dc\a1\ed\ab\1e\43\8f\37\3a\3f\7e\21\ed\10\d8\bc\16\99\74") ;; loop
    (data (i32.const 0x120)
        "\37\6b\42\13\e0\51\2e\29\5e\17\39\c1\40\33\f6\69\71\e9\92\ed\3d\6b\2f\3f\f6\cd\a7\b5\5f\97\e4\e3") ;; math
    (data (i32.const 0x140)
        "\8a\26\12\f2\89\05\cd\57\2a\c5\17\67\6a\0e\42\9e\3c\3b\7d\ca\d0\96\a9\54\95\b3\09\50\ca\6c\a8\bf") ;; memory
    (data (i32.const 0x160)
        "\4f\0f\58\40\8e\e9\68\21\2c\51\34\b3\e8\36\85\2a\53\5a\51\ba\96\0f\9a\04\30\5c\f1\24\70\ef\8f\3f") ;; grow
    (data (i32.const 0x180)
        "\4b\4f\41\14\49\ac\61\13\f2\95\7f\a1\4a\6c\48\00\ff\af\56\3e\69\f6\75\58\a7\3a\f9\1b\f1\e0\8d\a3") ;; return
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
