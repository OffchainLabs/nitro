;; Copyright 2023, Offchain Labs, Inc.
;; For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

(module
    (import "hostio" "wavm_link_module"        (func $link (param i32) (result i32)))
    (import "hostio" "wavm_unlink_module"      (func $unlink (param) (result)))
    (import "env" "wavm_halt_and_set_finished" (func $halt                         ))

    ;; WAVM module hashes
    (data (i32.const 0x000)
        "\3f\7f\62\83\74\ad\3a\e0\64\1d\a0\57\68\5d\62\45\75\24\52\74\c0\1d\a5\98\7d\88\07\a5\bd\5e\da\9f") ;; block
    (data (i32.const 0x020)
        "\05\96\08\cf\eb\1b\91\96\0c\20\e9\25\71\f3\7a\d6\c9\81\e8\d4\65\b1\92\15\28\84\44\f4\83\53\c9\cf") ;; call
    (data (i32.const 0x040)
        "\53\f8\9f\2b\4a\98\73\8e\1d\cd\6c\fb\ff\29\65\08\b7\4c\0b\0d\64\d3\b0\1c\e0\80\c4\11\f9\87\62\6c") ;; indirect
    (data (i32.const 0x060)
        "\e9\ad\a8\ba\75\15\79\2a\14\5b\89\b5\ce\4c\af\63\d7\49\48\68\31\11\be\56\b2\0c\cc\fc\ee\66\d1\c2") ;; const
    (data (i32.const 0x080)
        "\10\d9\a0\11\aa\bb\0a\3d\12\f9\09\d8\4f\23\9c\f4\c3\12\41\46\b4\1d\aa\b5\76\7f\83\34\8b\ee\59\e9") ;; div
    (data (i32.const 0x0a0)
        "\b1\5a\27\83\1c\ae\d2\64\71\c5\4c\76\72\78\ea\62\fa\2b\76\5e\10\fa\da\36\03\a2\bc\35\0e\2b\8f\14") ;; globals
    (data (i32.const 0x0c0)
        "\50\a8\58\3e\70\4e\e5\49\c6\83\db\04\e8\61\7d\6a\00\58\57\88\0c\0b\cf\0e\40\65\c0\fc\2f\ee\d7\25") ;; if-else
    (data (i32.const 0x0e0)
        "\2e\23\0f\99\b6\63\3c\87\e4\55\b9\2d\c6\9f\d2\48\29\4c\cc\af\a8\07\f7\99\49\5e\aa\32\1f\24\88\d2") ;; locals
    (data (i32.const 0x100)
        "\ec\e6\bc\36\7e\37\49\73\53\ee\be\23\dd\b9\97\52\b1\6f\2f\d3\e6\f7\c0\48\69\43\af\cd\5c\1f\52\df") ;; loop
    (data (i32.const 0x120)
        "\13\4f\e8\6f\7f\55\6c\cf\7a\56\6e\a7\0b\cb\7d\a4\a7\80\c3\62\74\29\58\a2\d6\2c\b0\15\9f\9a\9f\4c") ;; math
    (data (i32.const 0x140)
        "\e5\f4\9d\3c\4d\8c\62\5d\2f\e5\23\53\a9\4f\f0\1e\d7\72\07\e4\33\d7\8a\73\f6\bf\3f\0d\00\61\92\8f") ;; iops
    (data (i32.const 0x160)
        "\66\bd\f4\8a\70\5a\41\12\25\3f\77\1c\e0\66\8d\43\74\57\8d\f8\2e\17\dc\7e\8b\6c\98\49\9c\02\55\6d") ;; user
    (data (i32.const 0x180)
        "\1c\e5\83\c3\ff\6b\12\6c\34\b8\a3\d4\33\1c\3d\59\cb\5f\a1\60\4e\fb\10\02\19\4b\80\d4\15\b7\0a\97") ;; return

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
