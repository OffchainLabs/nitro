;; Copyright 2023, Offchain Labs, Inc.
;; For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

(module
    (import "hostio" "wavm_link_module"        (func $link (param i32) (result i32)))
    (import "hostio" "wavm_unlink_module"      (func $unlink (param) (result)))
    (import "env" "wavm_halt_and_set_finished" (func $halt                         ))

    ;; WAVM module hashes
    (data (i32.const 0x000)
        "\eb\12\b0\76\57\15\ad\16\0a\78\54\4d\c7\8d\d4\86\1c\58\a3\ee\77\f9\4a\4e\61\a3\f1\7f\d9\d2\be\8a") ;; block
    (data (i32.const 0x020)
        "\01\90\21\0c\1d\c8\45\9c\32\ef\a6\00\44\3b\e0\b6\31\70\1f\ce\7a\38\90\1c\e0\c5\40\6d\d8\ce\30\a6") ;; call
    (data (i32.const 0x040)
        "\e1\a2\fa\8e\81\2a\34\2e\cf\0f\62\46\ba\a4\45\8e\2d\95\2f\ec\1e\79\8f\dc\1b\1c\b8\15\cf\26\02\6c") ;; indirect
    (data (i32.const 0x060)
        "\ae\cb\eb\e9\0b\5e\1f\78\1b\66\5b\ff\8a\a4\18\a1\a2\e9\90\26\8b\df\df\95\64\54\82\07\6a\d4\e6\20") ;; const
    (data (i32.const 0x080)
        "\8b\7b\7e\a8\b8\21\c8\d0\2a\80\7c\1e\4b\6d\0d\07\f3\2d\8b\4e\f1\6b\e4\44\03\cf\05\66\9b\09\be\6d") ;; div
    (data (i32.const 0x0a0)
        "\da\4a\41\74\d6\2e\20\36\8e\cb\8e\5d\45\12\1c\28\1d\c4\8f\1d\77\92\9f\07\a8\6b\35\ea\89\2e\f9\72") ;; globals
    (data (i32.const 0x0c0)
        "\3f\ec\7c\06\04\b2\0d\99\bb\10\85\61\91\ea\b6\97\a7\a2\d1\19\67\2e\7c\d9\17\d4\6b\45\e8\4b\83\4b") ;; if-else
    (data (i32.const 0x0e0)
        "\30\12\24\71\df\9f\a9\f8\9c\33\9b\37\a7\08\f5\aa\5f\53\68\b4\e4\de\66\bb\73\ff\30\29\47\5f\50\e5") ;; locals
    (data (i32.const 0x100)
        "\f3\95\dd\a7\e1\d7\df\94\06\ca\93\0f\53\bf\66\ce\1a\aa\b2\30\68\08\64\b5\5b\61\54\2c\1d\62\e8\25") ;; loop
    (data (i32.const 0x120)
        "\8c\a3\63\7c\4e\70\f7\79\13\0c\9a\94\5e\63\3b\a9\06\80\9f\a6\51\0e\32\34\e0\9d\78\05\6a\30\98\0f") ;; math
    (data (i32.const 0x140)
        "\47\f7\4f\9c\21\51\4f\52\24\ea\d3\37\5c\bf\a9\1b\1a\5f\ef\22\a5\2a\60\30\c5\52\18\90\6b\b1\51\e5") ;; iops
    (data (i32.const 0x160)
        "\a1\49\cf\81\13\ff\9c\95\f2\c8\c2\a1\42\35\75\36\7d\e8\6d\d4\22\d8\71\14\bb\9e\a4\7b\af\53\5d\d7") ;; user
    (data (i32.const 0x180)
        "\ee\47\08\f6\47\b2\10\88\1f\89\86\e7\e3\79\6b\b2\77\43\f1\4e\ee\cf\45\4a\9b\7c\d7\c4\5b\63\b6\d7") ;; return

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
