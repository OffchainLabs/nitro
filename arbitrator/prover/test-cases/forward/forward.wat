
(module
    (import "target" "arbitrator_forward__add" (func $add (param i32 i32) (result i32)))
    (import "target" "arbitrator_forward__sub" (func $sub (param i32 i32) (result i32)))
    (import "target" "arbitrator_forward__mul" (func $mul (param i32 i32) (result i32)))
    (export "forward__add" (func $add))
    (export "forward__sub" (func $sub))
    (export "forward__mul" (func $mul)))
