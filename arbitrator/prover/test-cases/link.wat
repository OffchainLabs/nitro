;; Copyright 2023, Offchain Labs, Inc.
;; For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

(module
    (import "hostio" "link_module" (func $link (param i32) (result i32)))
    (import "hostio" "unlink_module" (func $unlink (param) (result)))
    (data (i32.const 0x000)
        "\44\c2\44\cf\34\9f\b0\8d\ad\8d\c8\24\b7\60\93\f5\b4\a9\29\5d\98\21\ff\d0\00\b1\88\11\14\fd\9a\6d") ;; call
    (data (i32.const 0x020)
        "\ad\e1\0c\24\4b\45\86\41\f8\99\34\19\2a\d4\39\95\55\bd\9b\e5\41\ce\33\be\bd\b8\21\6a\a2\7e\6e\35") ;; indirect
    (data (i32.const 0x040)
        "\86\99\7e\89\48\42\81\72\ce\64\14\d9\84\6d\9b\8e\e2\4b\d2\f7\ad\47\67\73\50\55\c3\f7\fb\a5\dc\b1") ;; const
    (data (i32.const 0x060)
        "\02\f2\68\25\d7\2d\11\02\94\d7\89\38\db\d4\b6\a4\3b\60\f7\8e\ae\2e\89\d2\a3\88\66\4c\65\3d\73\18") ;; div
    (data (i32.const 0x080)
        "\3f\a5\20\59\ae\19\da\10\a0\92\43\30\71\44\8d\ca\c1\4d\0d\aa\28\0c\d3\88\0f\2b\15\ab\df\14\a5\07") ;; globals
    (data (i32.const 0x0a0)
        "\f7\72\99\1c\3d\b0\ca\8f\96\b6\88\46\c2\f6\38\56\fe\e3\ca\c1\2b\f2\e2\d1\77\b6\5e\64\1f\67\46\d8") ;; if-else
    (data (i32.const 0x0c0)
        "\46\20\ed\2c\c4\6b\aa\dd\34\28\53\ba\ae\02\14\cb\44\1e\bc\63\cb\16\6f\50\c5\24\da\6a\e1\a0\33\32") ;; locals
    (data (i32.const 0x0e0)
        "\f7\0b\10\89\6d\0a\b7\36\82\a0\9e\39\9e\aa\69\b8\57\b6\78\65\ca\6a\a3\5d\81\40\40\77\23\b0\49\b7") ;; loop
    (data (i32.const 0x100)
        "\b8\14\b5\dd\ea\3c\4b\72\be\c0\e1\6e\ee\eb\ce\f2\70\0d\a7\57\fa\e8\21\db\9f\b2\02\b0\0d\7a\22\eb") ;; math
    (data (i32.const 0x120)
        "\ec\ff\3f\8c\5e\b1\9c\53\be\2d\e2\14\69\97\fe\91\f3\90\cb\9f\0b\a0\aa\df\ac\01\e6\dd\0e\d8\30\6b") ;; memory
    (data (i32.const 0x140)
        "\20\ca\af\79\ad\c9\fc\3a\dd\22\26\d2\64\9f\f1\ae\94\cd\02\40\22\e4\69\dc\b4\8c\15\ae\12\54\cf\31") ;; grow
    (data (i32.const 0x160)
        "\60\9c\2e\e9\45\19\7b\c1\82\06\cc\ff\40\61\1d\49\de\2a\66\b0\38\dc\00\b8\61\18\9f\c6\8f\7d\19\82") ;; return
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
