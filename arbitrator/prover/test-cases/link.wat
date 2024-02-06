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
        "\d1\2d\6d\d2\ec\c5\29\c2\c9\fa\d7\82\10\67\5e\d3\75\ea\75\5a\f8\b2\17\98\a3\99\db\7a\f1\e4\77\6a") ;; const
    (data (i32.const 0x080)
        "\fc\bc\04\84\5a\99\e2\77\f4\2d\eb\d2\79\b3\76\42\2b\1a\bd\4f\32\43\85\4b\78\2a\f8\4a\b9\00\c9\f1") ;; div
    (data (i32.const 0x0a0)
        "\22\59\23\96\83\94\1a\54\c9\e6\7b\cb\61\b8\e5\6c\4b\68\85\aa\0c\ae\2e\bc\e4\98\91\0e\69\c5\ab\88") ;; globals
    (data (i32.const 0x0c0)
        "\24\ca\89\ec\a2\3e\ea\45\88\82\f2\f5\af\5f\48\e3\39\8d\1a\d8\2d\53\a6\bb\64\0a\0c\9e\a9\79\0b\fc") ;; if
    (data (i32.const 0x0e0)
        "\66\e7\7d\41\50\76\ae\ce\7a\51\b5\6b\78\69\2e\b8\ab\24\79\a8\52\02\36\20\81\80\7e\17\0e\f3\da\fd") ;; locals
    (data (i32.const 0x100)
        "\f2\c7\18\ab\67\da\dd\5f\b5\7f\76\95\0d\00\eb\ca\0c\94\1f\aa\73\0d\b3\9e\90\5e\20\16\93\8b\fd\2a") ;; loop
    (data (i32.const 0x120)
        "\13\4f\e8\6f\7f\55\6c\cf\7a\56\6e\a7\0b\cb\7d\a4\a7\80\c3\62\74\29\58\a2\d6\2c\b0\15\9f\9a\9f\4c") ;; math
    (data (i32.const 0x140)
        "\46\ab\9d\c4\06\42\9f\57\81\d0\ea\71\67\f9\2a\6d\77\66\d0\16\1e\79\de\73\1e\14\78\bc\f3\7a\61\2a") ;; iops
    (data (i32.const 0x160)
        "\66\bd\f4\8a\70\5a\41\12\25\3f\77\1c\e0\66\8d\43\74\57\8d\f8\2e\17\dc\7e\8b\6c\98\49\9c\02\55\6d") ;; user
    (data (i32.const 0x180)
        "\9b\e0\8d\53\c2\98\45\d2\a4\a7\77\82\49\14\04\35\9d\42\01\78\dd\07\d3\6e\c8\1b\fe\c4\97\6e\56\c8") ;; return

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
