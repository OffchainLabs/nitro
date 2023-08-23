;; Copyright 2023, Offchain Labs, Inc.
;; For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

(module
    (import "hostio" "wavm_link_module" (func $link (param i32) (result i32)))
    (import "hostio" "wavm_unlink_module" (func $unlink (param) (result)))
    (data (i32.const 0x0)
         "\88\ff\33\e2\65\70\e0\88\b2\1f\03\64\34\36\05\7e\39\71\a5\6c\ba\96\76\7b\6a\e9\70\13\13\46\95\2f")
    (data (i32.const 0x20)
         "\d2\49\05\37\66\54\9e\fc\eb\af\cb\3d\50\71\2b\34\45\34\02\45\ed\16\83\34\bb\63\8b\c7\e6\a1\ff\10")
    (data (i32.const 0x40)
         "\e8\14\3b\94\37\e9\51\b6\58\41\40\77\8b\82\bf\c9\df\23\35\a1\74\9d\8c\0e\03\eb\5d\51\b0\13\5f\91")
    (data (i32.const 0x60)
         "\4f\bb\49\69\d5\5e\d7\bc\c8\15\4b\4d\44\47\2a\0d\99\c6\d0\6f\c4\45\12\b7\23\4d\08\7d\e5\d8\f3\90")
    (data (i32.const 0x80)
         "\e6\b9\67\33\7a\c5\b0\b5\76\00\3a\6e\f2\9b\11\2f\42\64\b1\ae\98\b1\77\92\b0\b1\51\58\23\94\d6\ee")
    (data (i32.const 0xa0)
         "\7f\96\bd\e6\06\55\44\38\ec\a9\82\e5\3c\0d\b2\76\b2\62\9d\20\91\65\c8\ff\ed\20\0e\59\7e\ef\38\a0")
    (data (i32.const 0xc0)
         "\36\7c\f6\0c\3c\bc\29\2f\ab\7d\4e\59\2c\6b\61\1d\c5\9c\49\a5\65\d3\a7\ef\2d\2a\f7\f1\d0\b1\5e\e9")
    (data (i32.const 0xe0)
         "\be\9e\03\4f\9e\57\a7\c4\ae\af\8f\43\65\55\8e\68\d7\81\1a\e9\07\4e\5e\a8\d1\3d\21\34\e4\18\dd\68")
    (data (i32.const 0x100)
         "\b0\9d\f3\19\d9\ac\bc\dd\cf\55\b0\7b\06\6d\98\2c\59\7d\07\88\47\b3\b2\22\ca\40\64\22\30\ae\a0\67")
    (data (i32.const 0x120)
         "\80\d3\d5\e6\1a\9a\9d\58\9a\e8\42\d5\69\2f\c2\38\16\47\44\b1\5b\66\c5\d6\dc\8f\f5\b3\66\91\4a\ee")
    (data (i32.const 0x140)
         "\db\ec\76\43\14\38\9b\f4\a6\6e\25\f7\f5\94\6c\da\b6\e4\f9\cc\e6\2f\00\36\bc\e9\8c\66\fd\dd\68\f9")
    (data (i32.const 0x160)
         "\2c\ab\68\c3\22\a0\7e\25\00\a5\64\5b\29\72\27\bb\c0\54\d1\60\69\9b\d9\c9\c8\7c\30\1e\eb\ea\3e\f9")
    (data (i32.const 0x180)
         "\ef\4e\09\6e\04\c1\9f\eb\58\84\a5\40\8a\79\27\23\ac\e8\bf\bf\db\be\e2\f3\cb\4c\c3\d2\5f\c7\c7\4c")
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
