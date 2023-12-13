;; Copyright 2023, Offchain Labs, Inc.
;; For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

(module
    (import "hostio" "wavm_link_module"        (func $link (param i32) (result i32)))
    (import "hostio" "wavm_unlink_module"      (func $unlink (param) (result)))
    (import "env" "wavm_halt_and_set_finished" (func $halt                         ))

    ;; WAVM module hashes
    (data (i32.const 0x000)
        "\56\e6\21\3d\17\66\a1\45\87\00\18\12\9f\bc\f3\64\14\78\38\2e\4a\46\e7\df\c7\77\68\4f\1d\8a\79\d7") ;; block
    (data (i32.const 0x020)
        "\dc\a8\84\65\df\1f\e5\d8\b2\e7\92\e0\23\9c\17\d0\9b\dd\bd\37\85\dc\a0\4b\3a\23\e5\fc\1a\83\fe\1a") ;; call
    (data (i32.const 0x040)
        "\46\d3\80\92\01\62\c0\91\66\d0\6a\50\69\d1\47\f8\60\07\bc\51\fd\36\a0\90\c2\1b\c7\5d\fe\ed\d9\28") ;; indirect
    (data (i32.const 0x060)
        "\cf\e5\4c\be\40\74\08\5f\9b\2f\2d\2f\f3\ec\29\02\4f\5a\d6\73\75\47\d0\23\2a\d3\97\fd\2e\92\a7\20") ;; const
    (data (i32.const 0x080)
        "\17\06\7b\73\ad\71\f4\9e\87\4b\03\2b\2d\8f\79\31\45\9c\5f\bd\a4\70\b4\b3\0f\ff\06\0e\0a\3e\f6\15") ;; div
    (data (i32.const 0x0a0)
        "\ad\e1\ab\3b\8a\06\a9\67\a6\ca\70\10\1e\88\eb\0e\76\4b\49\b9\13\db\6c\5a\a7\13\40\5f\a0\d8\8e\cf") ;; globals
    (data (i32.const 0x0c0)
        "\74\d8\18\a2\fd\74\bb\4f\8a\e4\06\e0\3b\07\36\39\fc\ec\a0\4f\1f\29\5e\24\b0\a2\13\bb\92\0c\6c\e4") ;; if-else
    (data (i32.const 0x0e0)
        "\95\3c\d1\ca\08\aa\97\38\e8\d0\ba\43\17\3c\4f\04\82\c8\1d\af\b1\03\da\f2\3a\31\f8\a8\da\0d\4a\14") ;; locals
    (data (i32.const 0x100)
        "\78\ac\75\14\39\23\92\49\6e\07\0a\82\5b\be\24\be\1b\c0\d8\4a\e8\33\64\0a\91\29\59\64\df\b1\02\86") ;; loop
    (data (i32.const 0x120)
        "\a8\73\72\41\8a\02\db\aa\19\6f\ec\ba\df\2d\09\a0\36\92\6f\d9\ee\d3\f0\63\6d\50\19\34\f0\15\18\91") ;; math
    (data (i32.const 0x140)
        "\7d\91\e4\b8\fa\48\8f\f5\24\70\57\3a\fd\f3\24\1a\6c\87\4d\9e\2a\4b\fd\17\48\dc\22\da\b9\a6\7c\66") ;; iops
    (data (i32.const 0x160)
        "\08\f6\70\9a\1b\e8\6f\84\85\f0\ff\fd\8d\f2\ef\77\74\dc\f5\a5\d7\d0\cb\a1\6b\a3\59\98\09\04\c7\5f") ;; user
    (data (i32.const 0x180)
        "\a2\85\7b\bd\ae\95\9e\f5\28\b3\dd\be\f9\70\c8\23\cb\76\83\e0\62\b9\9a\02\bc\71\02\ff\81\47\47\ad") ;; return

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
