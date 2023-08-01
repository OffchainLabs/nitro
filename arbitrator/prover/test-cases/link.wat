;; Copyright 2023, Offchain Labs, Inc.
;; For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

(module
    (import "hostio" "wavm_link_module" (func $link (param i32) (result i32)))
    (import "hostio" "wavm_unlink_module" (func $unlink (param) (result)))
    (data (i32.const 0x000)
         "\6b\56\5a\a3\ec\ca\cf\6f\83\96\c2\bb\c9\c3\c8\07\71\69\2b\0c\2a\a3\c6\d1\6d\24\d6\10\7f\4a\74\92")
    (data (i32.const 0x020)
         "\62\42\d1\93\7c\ee\c9\78\15\c1\bf\75\9a\2e\8e\c3\b5\16\23\26\3f\00\3f\bf\88\a0\c8\83\43\7f\9d\6a")
    (data (i32.const 0x040)
         "\cb\17\0a\20\c9\c8\12\3f\fd\e0\e1\77\c4\a7\c6\ac\aa\ad\c0\51\18\04\b3\66\4e\80\b0\1c\e6\30\39\0f")
    (data (i32.const 0x060)
         "\8f\7c\08\6b\39\9f\cf\a6\ed\e7\2a\0e\fc\7d\bc\90\58\17\28\6a\d1\7e\0a\02\e3\d8\ba\fc\59\d5\78\87")
    (data (i32.const 0x080)
         "\d8\71\08\f8\cb\ab\11\3b\6a\99\1e\0c\82\da\1c\ba\2a\f7\71\47\ac\b9\a0\ab\3c\c6\8b\8c\b7\95\b8\73")
    (data (i32.const 0x0a0)
         "\c5\27\90\19\f2\b6\5e\7b\5e\c7\2d\11\8f\71\37\a5\a2\47\61\4e\c3\bb\8d\49\88\0a\d9\52\c9\f5\aa\4c")
    (data (i32.const 0x0c0)
         "\c7\2c\02\ce\c5\f3\17\02\85\70\31\30\8d\53\d3\2c\82\1c\2d\4f\e1\1e\68\32\08\61\73\af\90\3f\c1\f8")
    (data (i32.const 0x0e0)
         "\20\db\c1\9a\5a\aa\63\da\47\26\f8\9c\a0\64\7b\f2\aa\93\39\d3\3c\f9\8a\0c\3e\38\8d\f2\6c\f2\d4\a9")
    (data (i32.const 0x100)
         "\d2\41\eb\7b\a0\df\33\59\aa\80\7e\0d\7f\9e\6d\91\74\34\71\4d\74\f1\5d\70\f9\10\f9\ce\c0\81\a9\a0")
    (data (i32.const 0x120)
         "\34\24\4c\84\9b\ad\81\38\9e\ae\1b\95\c0\f1\07\1f\4e\53\28\4c\13\2f\c0\34\7b\33\0f\3d\f3\f4\53\a3")
    (data (i32.const 0x140)
         "\95\d3\c7\9d\44\69\91\f1\a4\f2\fe\c9\99\b8\52\28\70\7b\f4\88\a4\9a\df\36\91\c0\17\7d\7b\76\ec\09")
    (data (i32.const 0x160)
         "\a8\14\b0\a9\a0\23\cc\ed\26\2a\cd\f0\98\b8\bb\d8\67\31\83\39\73\f7\31\ad\14\a4\3a\f9\7a\c6\b9\af")
    (data (i32.const 0x180)
         "\a7\ff\9f\75\57\98\78\14\f4\1f\de\4e\a9\48\ac\64\de\51\66\ce\28\6d\bc\22\17\80\83\00\0c\e2\99\5a")
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
