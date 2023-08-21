;; Copyright 2023, Offchain Labs, Inc.
;; For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

(module
    (import "hostio" "wavm_link_module" (func $link (param i32) (result i32)))
    (import "hostio" "wavm_unlink_module" (func $unlink (param) (result)))
    (data (i32.const 0x000)
        "\ae\dc\38\6b\36\2c\42\23\73\d0\10\e1\e6\72\a8\6b\58\9f\10\79\8f\3a\84\89\f1\c1\55\d8\3a\7c\74\08") ;; block
    (data (i32.const 0x020)
        "\a5\3a\c1\b1\a1\2e\87\f1\a9\68\67\13\25\1e\f9\75\85\30\5f\51\47\3c\87\3f\f4\4b\02\74\00\53\b7\44") ;; call
    (data (i32.const 0x040)
        "\ad\24\b8\29\6b\15\4e\50\48\0b\69\89\f1\cc\ed\68\22\ae\2f\b2\e8\3d\ed\50\06\d4\fb\5b\c1\bd\dd\e1") ;; indirect
    (data (i32.const 0x060)
        "\3d\92\82\57\c7\5f\03\cd\98\d1\49\7a\7b\6b\e1\13\b0\d3\92\38\94\f4\27\3b\5a\94\e4\2f\8c\ac\fb\06") ;; const
    (data (i32.const 0x080)
        "\27\6e\5d\0d\79\e8\b8\c5\e4\77\45\e4\8e\fb\93\eb\b9\83\1e\38\e1\a5\34\e5\15\a3\87\af\75\fc\b0\75") ;; div
    (data (i32.const 0x0a0)
        "\3f\b4\8c\32\cd\4e\12\1b\a6\af\18\d4\36\b2\2c\87\ba\f3\08\e9\d6\6d\91\61\69\dd\cc\91\6b\ae\77\6d") ;; globals
        "\31\81\c9\76\80\55\57\40\6d\93\0d\46\3b\60\31\de\4b\0f\93\14\8e\78\58\63\8c\66\88\55\c3\d3\47\b2") ;; if-else
        "\8f\b0\a8\9e\16\fa\76\ac\3e\16\86\94\4b\ce\17\e1\87\c6\ed\de\da\4d\49\9b\b4\70\47\7d\0b\0f\cf\c5") ;; if-else
    (data (i32.const 0x0e0)
        "\ec\2c\89\ff\20\c7\a8\af\4b\76\e0\0d\18\d7\24\27\aa\86\81\50\2a\f6\41\31\01\9f\24\fc\cf\06\92\b8") ;; locals
    (data (i32.const 0x100)
        "\f5\70\c9\95\e1\71\4b\55\fe\70\1f\90\ce\31\c4\ed\11\35\25\b0\4a\4d\01\f9\3c\77\39\8b\f4\cd\0c\10") ;; loop
    (data (i32.const 0x120)
        "\54\07\a2\84\19\02\c5\5c\3c\d9\52\3c\fd\03\7a\b3\d5\1b\00\b7\9a\89\cf\de\ed\5a\c0\69\90\31\49\0d") ;; math
    (data (i32.const 0x140)
        "\ea\15\0f\0e\ae\6d\e9\21\05\f4\45\bd\a8\b6\0f\4f\ea\e6\57\f4\b4\d5\64\e5\7e\bb\1b\6c\12\82\8a\77") ;; iops
    (data (i32.const 0x160)
        "\a6\d9\ac\fb\b4\01\cd\f8\4d\eb\6c\4c\07\cd\89\97\f7\c6\76\07\a7\6a\e9\a6\6f\60\04\c4\34\e7\2b\eb") ;; user
    (data (i32.const 0x180)
        "\1f\e6\67\ce\e9\86\70\06\b5\11\5d\fd\08\c1\6b\76\c3\8d\6c\a2\de\42\e5\ab\45\89\cc\6d\c0\88\d7\c4") ;; return
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
