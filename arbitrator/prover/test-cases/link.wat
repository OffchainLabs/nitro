;; Copyright 2023, Offchain Labs, Inc.
;; For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

(module
    (import "hostio" "wavm_link_module" (func $link (param i32) (result i32)))
    (import "hostio" "wavm_unlink_module" (func $unlink (param) (result)))
    (data (i32.const 0x0)
         "\6d\c0\9f\17\5f\5b\e8\73\64\bc\79\62\e8\13\fd\cb\09\2a\12\24\87\4a\af\15\f2\e1\2e\93\b0\95\30\9a")
    (data (i32.const 0x20)
         "\f5\6b\4c\c7\19\da\61\01\e4\e4\9a\f1\04\ca\29\97\fd\07\05\d6\c2\3b\e6\55\70\c5\54\65\a0\3f\3d\ee")
    (data (i32.const 0x40)
         "\57\27\40\77\40\da\77\f8\1f\fd\81\cb\00\e0\02\17\40\f0\be\e4\11\89\0a\56\ba\80\e4\b9\31\74\13\a2")
    (data (i32.const 0x60)
         "\53\36\71\e6\bf\90\0f\50\fd\18\5f\44\d6\18\77\2f\70\17\19\2a\1a\8d\b6\92\5a\3c\14\1a\af\86\81\d4")
    (data (i32.const 0x80)
         "\97\0c\df\6a\a9\bf\d4\3c\03\80\7f\8a\7e\67\9a\5c\12\05\94\4f\c6\5e\39\9e\00\df\5c\b3\7d\de\55\ad")
    (data (i32.const 0xa0)
         "\c7\db\9f\8e\ed\13\ac\66\72\62\76\65\93\1b\9a\64\03\c3\c8\21\44\92\5c\8d\bc\1a\d6\bd\65\f8\2b\20")
    (data (i32.const 0xc0)
         "\83\46\03\41\b4\5f\a6\e6\a3\0d\e9\fc\79\fc\3c\d6\c9\c3\c7\ac\97\42\bc\48\54\92\e6\84\08\37\07\a6")
    (data (i32.const 0xe0)
         "\42\1d\62\e9\9a\51\d4\71\ce\50\6e\b4\83\72\18\ea\f8\ab\ab\b9\29\b8\bd\6d\66\ea\52\b3\3d\50\26\34")
    (data (i32.const 0x100)
         "\74\22\43\ad\22\2e\e5\6d\f4\bb\3f\0b\09\76\0a\bf\51\b7\17\a4\c5\50\c9\5b\45\be\ea\ed\4c\57\4d\17")
    (data (i32.const 0x120)
         "\16\90\98\f2\7f\8d\bf\73\90\b9\eb\94\9f\b9\41\cd\c3\93\2e\30\b8\12\1b\d5\87\98\18\26\f2\62\7d\2c")
    (data (i32.const 0x140)
         "\3f\c3\a1\eb\a6\62\70\2b\3b\fa\dc\5b\29\22\11\6f\58\4a\6e\e5\70\60\6f\cf\6c\66\d8\c9\77\c5\c9\23")
    (data (i32.const 0x160)
         "\a7\66\cb\0e\c4\31\ea\16\fd\c6\2f\d3\11\ca\4a\78\f8\48\6a\69\0a\4c\b9\1c\fc\47\f8\b6\63\6d\80\fa")
    (data (i32.const 0x180)
         "\ea\02\78\f7\a3\b3\e0\0e\55\f6\8f\13\87\d6\6f\04\38\b3\6b\4c\d5\33\e2\3d\0b\36\71\9f\57\f5\f0\59")
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
