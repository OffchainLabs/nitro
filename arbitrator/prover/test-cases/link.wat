;; Copyright 2023, Offchain Labs, Inc.
;; For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

(module
    (import "hostio" "link_module" (func $link (param i32) (result i32)))
    (import "hostio" "unlink_module" (func $unlink (param) (result)))
    (data (i32.const 0x000)
        "\f0\61\ee\61\5e\1b\f7\44\7a\00\7d\fa\72\d4\d1\ef\de\2e\e9\53\a5\5f\44\df\3c\0a\b9\91\0b\db\48\6a") ;; block
    (data (i32.const 0x020)
        "\90\c1\a8\6b\a2\f5\06\06\93\47\b9\d5\63\88\12\0b\88\1e\e2\92\e8\be\fa\f5\7e\8f\1a\c2\70\9f\0e\d0") ;; call
    (data (i32.const 0x040)
        "\5e\fa\e0\2f\08\fb\68\60\ef\4e\69\db\7a\c1\e0\1d\09\56\1f\4e\c7\60\55\83\bf\7e\0d\91\02\d0\02\5e") ;; indirect
    (data (i32.const 0x060)
        "\2e\93\8c\b8\68\90\d3\5e\01\8b\cc\e9\05\5d\dc\5e\69\f3\32\41\7c\43\72\4b\5c\82\48\d6\06\18\a4\3b") ;; const
    (data (i32.const 0x080) 
        "\ad\c9\1c\82\09\d4\c3\12\1a\01\db\5c\38\f3\7a\a5\70\d1\b3\21\39\fc\60\c2\9f\79\a5\23\e0\e4\39\71") ;; div
    (data (i32.const 0x0a0)
        "\86\91\43\c1\91\b3\ff\f9\37\54\fc\90\9f\bf\29\07\38\ae\fa\be\0e\8c\99\45\68\5c\33\62\07\3f\f1\35") ;; globals
    (data (i32.const 0x0c0)
        "\8f\f3\bc\b4\55\07\17\c8\89\67\85\4a\53\fa\e6\31\b2\56\4e\c6\7e\1c\fd\08\2a\5f\24\c4\03\d2\33\25") ;; if-else
    (data (i32.const 0x0e0)
        "\25\52\43\00\80\6c\49\13\98\3d\c1\fb\40\81\32\5b\03\c1\15\30\5d\fd\71\92\0d\fc\91\43\58\0d\5a\2e") ;; locals
    (data (i32.const 0x100)
        "\f8\ef\b0\7c\70\6a\e8\d6\b2\a7\a6\50\ad\c1\68\87\32\61\c8\30\f0\c3\ff\33\8d\eb\49\82\1a\9c\5c\54") ;; loop
    (data (i32.const 0x120)
        "\63\bd\3f\6b\5f\b5\78\cf\63\36\59\39\4d\b8\ca\50\02\ad\be\d4\62\f2\14\59\e1\6f\7f\16\6d\47\78\87") ;; math
    (data (i32.const 0x140)
        "\f6\f0\c3\90\2a\b7\f6\b0\11\d5\9a\86\27\2f\5c\36\dc\8d\82\1a\5c\10\b7\6d\f8\a9\2b\fe\50\d2\9c\65") ;; memory
    (data (i32.const 0x160)
        "\45\b5\50\33\31\c7\d7\19\90\8d\97\60\7c\a3\a0\f2\aa\a0\2d\37\fc\d7\bd\3f\dc\78\a5\5f\a4\20\ad\b2") ;; grow
    (data (i32.const 0x180)
        "\d8\ec\96\4c\45\9b\f4\77\97\c3\d4\96\94\34\24\0d\2a\23\72\79\34\6c\ad\20\d2\02\64\c7\6b\4e\a7\40") ;; return
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
