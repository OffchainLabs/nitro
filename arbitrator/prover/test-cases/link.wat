
(module
    (import "hostio" "link_module" (func $link (param i32) (result i32)))
    (import "hostio" "unlink_module" (func $unlink (param) (result)))
    (data (i32.const 0x000)
        "\54\10\ee\94\52\f8\b3\cc\35\d0\eb\f8\7a\00\01\b8\f8\0d\1c\d5\16\d2\06\d6\09\ca\01\03\4e\66\6e\80") ;; block
    (data (i32.const 0x020)
        "\50\cf\6e\c1\05\84\58\32\ed\6b\ca\47\85\da\3a\74\cf\e0\c4\67\63\47\50\2e\a0\c4\10\1d\c6\75\48\af") ;; call
    (data (i32.const 0x040)
        "\da\70\31\76\0c\dc\98\a0\9c\c6\fb\5b\47\e6\a7\44\bc\a4\2d\be\03\54\fb\82\e5\0f\87\8f\8f\47\3b\11") ;; indirect
    (data (i32.const 0x060)
        "\48\d7\65\6f\2f\0c\27\40\d4\61\2c\30\a1\6c\1d\dc\f4\78\8c\c7\9a\77\c2\9c\ab\b1\2a\6d\c3\43\7c\8c") ;; const
    (data (i32.const 0x080) 
        "\fa\85\51\b4\b1\97\e4\85\60\37\71\82\7e\6c\53\1b\1c\a9\5f\37\77\72\f8\be\bb\aa\cf\9c\52\02\6b\45") ;; div
    (data (i32.const 0x0a0)
        "\84\10\70\b5\13\fa\91\d3\44\84\24\c9\b1\79\ac\7a\2b\09\56\4d\d1\e6\6d\87\cc\82\85\4c\02\f1\f5\12") ;; globals
    (data (i32.const 0x0c0)
        "\98\38\fc\02\31\8b\59\c7\f1\aa\1f\5c\5a\18\e1\f0\89\06\8a\db\40\de\78\b0\da\06\61\83\76\57\a4\dd") ;; if-else
    (data (i32.const 0x0e0)
        "\aa\ca\6f\03\40\24\26\0c\1f\0b\cb\f2\fc\3c\7d\b1\d4\f3\84\95\b5\fd\d5\0b\d2\ee\2b\df\ba\b0\43\90") ;; locals
    (data (i32.const 0x100)
        "\0d\f2\3d\0f\a6\d2\02\5a\c1\ae\93\98\f9\f9\7a\68\e8\2f\8c\0d\d2\a9\b6\5e\8a\ac\ad\6b\69\9a\f8\69") ;; loop
    (data (i32.const 0x120)
        "\8c\30\89\ff\89\52\64\e1\92\dd\e0\ff\bd\3d\17\9d\0d\b9\ee\19\d5\29\8b\ee\5b\b7\af\b8\99\5c\9c\8e") ;; math
    (data (i32.const 0x140)
        "\ed\09\f1\c4\ed\66\56\85\cb\ba\66\40\c1\81\ca\5b\5c\68\12\69\c1\9b\0b\5f\9e\b8\8f\d5\53\ec\82\5e") ;; memory
    (data (i32.const 0x160)
        "\95\03\fa\9a\18\31\93\40\b7\38\55\41\e5\ce\f1\88\71\21\b2\75\8c\08\68\36\45\51\04\07\c0\04\bd\1f") ;; grow
    (data (i32.const 0x180)
        "\cd\c9\4b\c7\a6\01\b7\d7\47\ab\e4\6e\01\cc\07\b9\db\f9\3b\6e\08\55\14\93\ef\af\1e\ba\be\34\40\b8") ;; return
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
